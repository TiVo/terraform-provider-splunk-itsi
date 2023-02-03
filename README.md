# terraform-provider-splunk-itsi
![Tests](https://github.com/tivo/terraform-provider-splunk-itsi/workflows/Tests/badge.svg?event=push)

Terraform Provider for Splunk ITSI
- [Provider documentation](https://registry.terraform.io/providers/TiVo/splunk-itsi/latest/docs)
- [ITSI REST API](https://docs.splunk.com/Documentation/ITSI/4.4.1/RESTAPI/ITSIRESTAPIreference)

## Prerequisites
- install Terraform (v.1.0.4): https://learn.hashicorp.com/tutorials/terraform/install-cli
- golang version 1.18 installed
- goreleaser: https://goreleaser.com/install/

#### Starting a provider in Debug Mode: 
It is important to start a provider in debug mode only when you intend to debug it, as its behavior will change in minor ways from normal operation of providers. The main differences are:
- Terraform will not start the provider process; it must be run manually.
- The provider's constraints will no longer be checked as part of terraform init.
- The provider will no longer be restarted once per walk of the Terraform graph; instead the same provider process will be reused until the command is completed.
### GoLand
Prerequisites:
- Go installed on your system
- <a href="https://www.jetbrains.com/go/download/#section=linux">GoLand</a> is installed

Steps:
- Pass debug flag, It can be done via the Run Configuration: <p><img src="./attachments/GoLand_RunConfiguration.png" alt="run_configuration" width="600"/></p>
- Make sure plugin is compiled in debug mode (`go build -gcflags="all=-N -l"`, GoLand takes care of this, executing via Debug button)
### VS Code
Prerequisites:
- Go installed on your system
- VS Code v1.74 installed on your computer
- <a href="https://code.visualstudio.com/docs/languages/go">Go</a> is installed in your VS Code editor

Steps:
- create a .vscode dir in the root of your clone terraform-provider-splunk-itsi local repo then create a launch.json inside that .vscode directory.
```
$ cat << 'EOF' > terraform-provider-splunk-itsi/.vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Provider",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}",
            "env": {"PKG_NAME": "${relativeFileDirname}"}, 
            "args": [
                "--debug",
            ],
            "showLog": true,
        }
    ]
}
EOF
```

### Common steps:
- After launching the plugin will output a message telling you to set the TF_REATTACH_PROVIDERS environment variable: <p><img src="./attachments/console_output.png" alt="console_output" width="600"/></p>
- Copy and paste this to another shell, from which you will run Terraform: <p><img src="./attachments/run_terraform_command.png" alt="run_terraform" width="1200"/></p>
- Observe your caught breakpoint: <p><img src="./attachments/breakpoint.png" alt="breakpoint" width="1200"/></p>


###### Useful links:
- https://opencredo.com/blogs/running-a-terraform-provider-with-a-debugger/
- https://www.youtube.com/watch?v=hwXXgd96bgA
- https://www.terraform.io/plugin/sdkv2/debugging

