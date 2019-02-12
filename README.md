# Cloud

## Usage:
```text
Cloudctl is a fast and flexible cloud development utility for multiple platforms

Author: Coleman Word
Download: gp get github.com/gofunct/cloud/...

 oooooooo8 o888                              oooo             o8   o888 
o888     88  888   ooooooo  oooo  oooo   ooooo888   ooooooo  o888oo  888 
888          888 888     888 888   888 888    888 888     888 888    888 
888o     oo  888 888     888 888   888 888    888 888         888    888 
 888oooo88  o888o  88ooo88    888o88 8o  88ooo888o  88ooo888   888o o888o

Usage:
  cloudctl [command]

Available Commands:
  debug       Common debugging operations |flags|config|
  help        Help about any command
  init        A brief description of your command

Flags:
      --bucket string         blob storage bucket (default "defbucket")
      --config string         config file (default is $PWD/cloudctl.yaml)
      --dbhost string         database host
      --dbname string         database name (default "defdb")
      --dbpass string         database password (default "defpass")
      --dbuser string         database user name (default "defuser")
  -h, --help                  help for cloudctl
      --port string           port to run app on (default ":8080")
      --runvar string         runtime variable value (default "hello world")
      --runvarname string     runtime variable name (default "motd")
      --runvarwait duration   timeout for runtime config watcher (default 30s)
      --sqlregion string      cloud sql region

Use "cloudctl [command] --help" for more information about a command.

```
* 