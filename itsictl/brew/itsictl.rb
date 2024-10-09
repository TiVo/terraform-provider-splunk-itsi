class Itsictl < Formula
  desc "Command-line tool to manage Splunk IT Service Intelligence (ITSI)"
  homepage "https://github.com/tivo/terraform-provider-splunk-itsi"
  version "v2.2.0"

  url "https://github.com/tivo/terraform-provider-splunk-itsi.git",
    branch: "main" #,
    #tag: "v2.2.0-beta.5",
    #revision: "f5cd5a3120a1617714e30faffcf5fe4795caa1f8"

  license "Apache-2.0"

  head "https://github.com/tivo/terraform-provider-splunk-itsi.git", branch: "main"

  depends_on "mise" => :build

  def install
    system "mise", "install", "--yes"
    system "mise", "exec", "--", "goreleaser", "build", "--single-target", "--snapshot", "--clean"
    bin.install Dir["dist/itsictl*/itsictl*"].first => "itsictl"

    # Generate the manpages using the itsictl genman command
    system bin/"itsictl", "genman"
    # Install all manpages from the 'man' directory
    man1.install Dir["man/*.1"]

    # Generate and install shell completions
    # Bash completion
    output = Utils.safe_popen_read(bin/"itsictl", "completion", "bash")
    (bash_completion/"itsictl").write output

    # Zsh completion
    output = Utils.safe_popen_read(bin/"itsictl", "completion", "zsh")
    (zsh_completion/"_itsictl").write output

    # Fish completion
    output = Utils.safe_popen_read(bin/"itsictl", "completion", "fish")
    (fish_completion/"itsictl.fish").write output
  end

end
