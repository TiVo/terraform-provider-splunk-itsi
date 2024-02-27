package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var validateStringID = struct {
	MinLength, MaxLength int
	RE                   *regexp.Regexp
	RegexpDescription    string
}{
	1, 255,
	regexp.MustCompile(`^[a-zA-Z][-_0-9a-zA-Z]*$`),
	"must begin with a letter and contain only alphanumerics, hyphens and underscores",
}

func validateStringIdentifier() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(validateStringID.MinLength, validateStringID.MaxLength),
		stringvalidator.RegexMatches(
			validateStringID.RE,
			validateStringID.RegexpDescription,
		),
	}
}
