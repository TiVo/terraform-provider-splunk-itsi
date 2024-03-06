package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// FmtCommand is a Command implementation that rewrites Terraform config
// files to a canonical format and style.
type FmtCommand struct {
	list      bool
	write     bool
	diff      bool
	check     bool
	recursive bool
	paths     []string
}

func NewFmtCommand(paths []string, check bool) *FmtCommand {
	cmd := &FmtCommand{
		list:      true,
		write:     true,
		diff:      false,
		check:     check,
		recursive: true,
		paths:     paths,
	}
	if cmd.check {
		// set to true so we can use the list output to check
		// if the input needs formatting
		cmd.list = true
		cmd.write = false
	}
	return cmd
}

func (c *FmtCommand) Run() diag.Diagnostics {

	output := &bytes.Buffer{}

	diags := c.fmt(c.paths, output)
	if diags.HasError() {
		return diags
	}

	if c.check {
		ok := output.Len() == 0
		if c.list {
			io.Copy(os.Stdout, output)
		}
		if ok {
			return nil
		} else {
			diags = append(diags, diag.Errorf(output.String())...)
		}
	}

	return diags
}

func (c *FmtCommand) fmt(paths []string, stdout io.Writer) diag.Diagnostics {
	var diags diag.Diagnostics

	if len(paths) == 0 {
		return diags
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			diags = append(diags, diag.Errorf("No file or directory at %s", path)...)
			return diags
		}
		if info.IsDir() {
			dirDiags := c.processDir(path, stdout)
			diags = append(diags, dirDiags...)
		} else {
			switch filepath.Ext(path) {
			case ".tf", ".tfvars":
				f, err := os.Open(path)
				if err != nil {
					// Open does not produce error messages that are end-user-appropriate,
					// so we'll need to simplify here.
					diags = append(diags, diag.Errorf("Failed to read file %s", path)...)
					continue
				}

				fileDiags := c.processFile(path, f, stdout, false)
				diags = append(diags, fileDiags...)
				f.Close()
			default:
				diags = append(diags, diag.Errorf("Only .tf and .tfvars files can be processed with terraform fmt")...)
				continue
			}
		}
	}

	return diags
}

func (c *FmtCommand) processFile(path string, r io.Reader, w io.Writer, isStdout bool) diag.Diagnostics {
	var diags diag.Diagnostics

	log.Printf("[TRACE] terraform fmt: Formatting %s", path)

	src, err := io.ReadAll(r)
	if err != nil {
		diags = append(diags, diag.Errorf("Failed to read %s", path)...)
		return diags
	}

	// // Register this path as a synthetic configuration source, so that any
	// // diagnostic errors can include the source code snippet
	// c.registerSynthConfigSource(path, src)

	// File must be parseable as HCL native syntax before we'll try to format
	// it. If not, the formatter is likely to make drastic changes that would
	// be hard for the user to undo.
	_, syntaxDiags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if syntaxDiags.HasErrors() {
		for _, err := range syntaxDiags.Errs() {
			diags = append(diags, diag.FromErr(err)...)
		}
		return diags
	}

	result := c.formatSourceCode(src, path)

	if !bytes.Equal(src, result) {
		// Something was changed
		if c.list {
			fmt.Fprintln(w, path)
		}
		if c.write {
			err := os.WriteFile(path, result, 0644)
			if err != nil {
				diags = append(diags, diag.Errorf("Failed to write %s", path)...)
				return diags
			}
		}
		if c.diff {
			diff, err := bytesDiff(src, result, path)
			if err != nil {
				diags = append(diags, diag.Errorf("Failed to generate diff for %s: %s", path, err)...)
				return diags
			}
			w.Write(diff)
		}
	}

	if !c.list && !c.write && !c.diff {
		_, err = w.Write(result)
		if err != nil {
			diags = append(diags, diag.Errorf("Failed to write result")...)
		}
	}

	return diags
}

func (c *FmtCommand) processDir(path string, stdout io.Writer) diag.Diagnostics {
	var diags diag.Diagnostics

	log.Printf("[TRACE] terraform fmt: looking for files in %s", path)

	entries, err := os.ReadDir(path)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			diags = append(diags, diag.Errorf("There is no configuration directory at %s", path)...)
		default:
			// ReadDir does not produce error messages that are end-user-appropriate,
			// so we'll need to simplify here.
			diags = append(diags, diag.Errorf("Cannot read directory %s", path)...)
		}
		return diags
	}

	for _, info := range entries {
		name := info.Name()
		// if configs.IsIgnoredFile(name) {
		// 	continue
		// }
		subPath := filepath.Join(path, name)
		if info.IsDir() {
			if c.recursive {
				subDiags := c.processDir(subPath, stdout)
				diags = append(diags, subDiags...)
			}

			// We do not recurse into child directories by default because we
			// want to mimic the file-reading behavior of "terraform plan", etc,
			// operating on one module at a time.
			continue
		}

		ext := filepath.Ext(name)
		switch ext {
		case ".tf", ".tfvars":
			f, err := os.Open(subPath)
			if err != nil {
				// Open does not produce error messages that are end-user-appropriate,
				// so we'll need to simplify here.
				diags = append(diags, diag.Errorf("Failed to read file %s", subPath)...)
				continue
			}

			fileDiags := c.processFile(subPath, f, stdout, false)
			diags = append(diags, fileDiags...)
			f.Close()
		}
	}

	return diags
}

// formatSourceCode is the formatting logic itself, applied to each file that
// is selected (directly or indirectly) on the command line.
func (c *FmtCommand) formatSourceCode(src []byte, filename string) []byte {
	f, diags := hclwrite.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		// It would be weird to get here because the caller should already have
		// checked for syntax errors and returned them. We'll just do nothing
		// in this case, returning the input exactly as given.
		return src
	}

	c.formatBody(f.Body(), nil)

	return f.Bytes()
}

func (c *FmtCommand) formatBody(body *hclwrite.Body, inBlocks []string) {
	attrs := body.Attributes()
	for name, attr := range attrs {
		if len(inBlocks) == 1 && inBlocks[0] == "variable" && name == "type" {
			cleanedExprTokens := c.formatTypeExpr(attr.Expr().BuildTokens(nil))
			body.SetAttributeRaw(name, cleanedExprTokens)
			continue
		}
		cleanedExprTokens := c.formatValueExpr(attr.Expr().BuildTokens(nil))
		cleanedExprTokens = c.formatValueMultiline(cleanedExprTokens)
		body.SetAttributeRaw(name, cleanedExprTokens)
	}

	blocks := body.Blocks()
	for _, block := range blocks {
		// Normalize the label formatting, removing any weird stuff like
		// interleaved inline comments and using the idiomatic quoted
		// label syntax.
		block.SetLabels(block.Labels())

		inBlocks := append(inBlocks, block.Type())
		c.formatBody(block.Body(), inBlocks)
	}
}

func (c *FmtCommand) formatValueMultiline(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) > 2 &&
		tokens[0].Type == hclsyntax.TokenOQuote &&
		tokens[len(tokens)-1].Type == hclsyntax.TokenCQuote {
		val := string(tokens[1 : len(tokens)-1].Bytes())
		if newval := strings.Split(val, "\\n"); len(newval) > 1 {
			attr_raw := []*hclwrite.Token{
				{
					Bytes: []byte("<<-EOT\n"),
					Type:  hclsyntax.TokenOHeredoc,
				},
			}
			for _, token := range newval {
				unescapedStr, err := strconv.Unquote(`"` + token + `"`)
				if err != nil {
					return tokens
				}
				attr_raw = append(attr_raw, &hclwrite.Token{
					Bytes: []byte(unescapedStr + "\n"),
					Type:  hclsyntax.TokenStringLit,
				})
			}
			return append(attr_raw,
				&hclwrite.Token{
					Bytes: []byte("EOT\n"),
					Type:  hclsyntax.TokenOHeredoc,
				})
		}
	}
	return tokens

}

func (c *FmtCommand) formatValueExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) < 5 {
		// Can't possibly be a "${ ... }" sequence without at least enough
		// tokens for the delimiters and one token inside them.
		return tokens
	}
	oQuote := tokens[0]
	oBrace := tokens[1]
	cBrace := tokens[len(tokens)-2]
	cQuote := tokens[len(tokens)-1]
	if oQuote.Type != hclsyntax.TokenOQuote || oBrace.Type != hclsyntax.TokenTemplateInterp || cBrace.Type != hclsyntax.TokenTemplateSeqEnd || cQuote.Type != hclsyntax.TokenCQuote {
		// Not an interpolation sequence at all, then.
		return tokens
	}

	inside := tokens[2 : len(tokens)-2]

	// We're only interested in sequences that are provable to be single
	// interpolation sequences, which we'll determine by hunting inside
	// the interior tokens for any other interpolation sequences. This is
	// likely to produce false negatives sometimes, but that's better than
	// false positives and we're mainly interested in catching the easy cases
	// here.
	quotes := 0
	for _, token := range inside {
		if token.Type == hclsyntax.TokenOQuote {
			quotes++
			continue
		}
		if token.Type == hclsyntax.TokenCQuote {
			quotes--
			continue
		}
		if quotes > 0 {
			// Interpolation sequences inside nested quotes are okay, because
			// they are part of a nested expression.
			// "${foo("${bar}")}"
			continue
		}
		if token.Type == hclsyntax.TokenTemplateInterp || token.Type == hclsyntax.TokenTemplateSeqEnd {
			// We've found another template delimiter within our interior
			// tokens, which suggests that we've found something like this:
			// "${foo}${bar}"
			// That isn't unwrappable, so we'll leave the whole expression alone.
			return tokens
		}
		if token.Type == hclsyntax.TokenQuotedLit {
			// If there's any literal characters in the outermost
			// quoted sequence then it is not unwrappable.
			return tokens
		}
	}

	// If we got down here without an early return then this looks like
	// an unwrappable sequence, but we'll trim any leading and trailing
	// newlines that might result in an invalid result if we were to
	// naively trim something like this:
	// "${
	//    foo
	// }"
	trimmed := c.trimNewlines(inside)

	// Finally, we check if the unwrapped expression is on multiple lines. If
	// so, we ensure that it is surrounded by parenthesis to make sure that it
	// parses correctly after unwrapping. This may be redundant in some cases,
	// but is required for at least multi-line ternary expressions.
	isMultiLine := false
	hasLeadingParen := false
	hasTrailingParen := false
	for i, token := range trimmed {
		switch {
		case i == 0 && token.Type == hclsyntax.TokenOParen:
			hasLeadingParen = true
		case token.Type == hclsyntax.TokenNewline:
			isMultiLine = true
		case i == len(trimmed)-1 && token.Type == hclsyntax.TokenCParen:
			hasTrailingParen = true
		}
	}
	if isMultiLine && !(hasLeadingParen && hasTrailingParen) {
		wrapped := make(hclwrite.Tokens, 0, len(trimmed)+2)
		wrapped = append(wrapped, &hclwrite.Token{
			Type:  hclsyntax.TokenOParen,
			Bytes: []byte("("),
		})
		wrapped = append(wrapped, trimmed...)
		wrapped = append(wrapped, &hclwrite.Token{
			Type:  hclsyntax.TokenCParen,
			Bytes: []byte(")"),
		})

		return wrapped
	}

	return trimmed
}

func (c *FmtCommand) formatTypeExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	switch len(tokens) {
	case 1:
		kwTok := tokens[0]
		if kwTok.Type != hclsyntax.TokenIdent {
			// Not a single type keyword, then.
			return tokens
		}

		// Collection types without an explicit element type mean
		// the element type is "any", so we'll normalize that.
		switch string(kwTok.Bytes) {
		case "list", "map", "set":
			return hclwrite.Tokens{
				kwTok,
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("any"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			return tokens
		}

	case 3:
		// A pre-0.12 legacy quoted string type, like "string".
		oQuote := tokens[0]
		strTok := tokens[1]
		cQuote := tokens[2]
		if oQuote.Type != hclsyntax.TokenOQuote || strTok.Type != hclsyntax.TokenQuotedLit || cQuote.Type != hclsyntax.TokenCQuote {
			// Not a quoted string sequence, then.
			return tokens
		}

		// Because this quoted syntax is from Terraform 0.11 and
		// earlier, which didn't have the idea of "any" as an,
		// element type, we use string as the default element
		// type. That will avoid oddities if somehow the configuration
		// was relying on numeric values being auto-converted to
		// string, as 0.11 would do. This mimicks what terraform
		// 0.12upgrade used to do, because we'd found real-world
		// modules that were depending on the auto-stringing.)
		switch string(strTok.Bytes) {
		case "string":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
			}
		case "list":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("list"),
				},
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		case "map":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("map"),
				},
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			// Something else we're not expecting, then.
			return tokens
		}
	default:
		return tokens
	}
}

func (c *FmtCommand) trimNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return nil
	}
	var start, end int
	for start = 0; start < len(tokens); start++ {
		if tokens[start].Type != hclsyntax.TokenNewline {
			break
		}
	}
	for end = len(tokens); end > 0; end-- {
		if tokens[end-1].Type != hclsyntax.TokenNewline {
			break
		}
	}
	return tokens[start:end]
}

func bytesDiff(b1, b2 []byte, path string) (data []byte, err error) {
	f1, err := os.CreateTemp("", "")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := os.CreateTemp("", "")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "--label=old/"+path, "--label=new/"+path, "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}
