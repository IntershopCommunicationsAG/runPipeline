Client to start an Azure DevOps Pipeline
========================================

This program starts an Azure DevOps Pipeline the pipeline remotely from commandline over REST.
The program finished if the pipeline finished and the exit code can be used for further utilization.

Parameter
---------

| Parameter                |          | usage                                                                                                                                                                            |
|--------------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| org <organization>       | required | This is the used Azure DevOps organization.                                                                                                                                      |
| prj <project>            | required | This is the used Azure DevOps project in the organization                                                                                                                        |
| token <PAT>              | required | Personal access token for login, see [Microsoft documentation](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate). |
| pipeline <pipeline name> | required | The name of the pipeline, that should be executed.                                                                                                                               |
| branch <branch name>     | optional | The name of the branch for pipeline. Default is 'master'.                                                                                                                        |
| param <key=value>        | optional | Parameters for the pipeline execution, eg. --param key1=value1.                                                                                                                  |
| w                        | optional | Warn log is enabled.                                                                                                                                                             |
| i                        | optional | Info log is enabled.                                                                                                                                                             |
| v                        | optional | Verbose log is enabled.                                                                                                                                                          |
| h                        | optional | Shows usage of the command.                                                                                                                                                      |

Usage
-----
```
Usage of ./runPipeline:
  -org string
        Azure DevOps organization.
  -prj string
        Azure DevOps project.
  -token string
        Azure DevOps personal access token
  -pipeline string
        Azure DevOps pipeline name
  -branch string
        Branch for pipeline run (default "master")
  -param value
        Parameter as string like 'key=value'
  -w    Logging with warn output
  -i    Logging with info output
  -v    Logging with verbose output
  -h    Shows usage of this command.
```




