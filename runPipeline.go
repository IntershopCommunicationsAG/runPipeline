/*
 * Copyright 2022 Intershop Communications AG.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/pipelines"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

const ADOURL = "https://dev.azure.com/%s"

type App struct {
	org        string
	prj        string
	token      string
	pipeline   string
	branch     string
	parameters []string

	infoLog    bool
	verboseLog bool
	warnLog    bool
}

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var paramsSlice stringSlice

func (app *App) ParseCommandLine() {
	paramOrgString := flag.String("org", "", "Azure DevOps organization.")
	paramPrjString := flag.String("prj", "", "Azure DevOps project.")
	paramTokenString := flag.String("token", "", "Azure DevOps personal access token")
	paramPipelineString := flag.String("pipeline", "", "Azure DevOps pipeline name")
	paramBranchString := flag.String("branch", "master", "Branch for pipeline run")
	flag.Var(&paramsSlice, "param", "Parameter as string like 'key=value'")
	paramVerboseOutput := flag.Bool("v", false, "Logging with verbose output")
	paramInfoOutput := flag.Bool("i", false, "Logging with info output")
	paramWarnOutput := flag.Bool("w", false, "Logging with warn output")

	paramHelp := flag.Bool("h", false, "Shows usage of this command.")

	showUsage()
	flag.Parse()

	if *paramHelp {
		flag.CommandLine.Usage()
		os.Exit(0)
	}

	if *paramOrgString == "" {
		fmt.Fprintln(os.Stderr, "Parameter 'org' is empty.")
		flag.CommandLine.Usage()
		os.Exit(1)
	}
	if *paramPrjString == "" {
		fmt.Fprintln(os.Stderr, "Parameter 'prj' is empty.")
		flag.CommandLine.Usage()
		os.Exit(2)
	}
	if *paramTokenString == "" {
		fmt.Fprintln(os.Stderr, "Parameter 'token' is empty.")
		flag.CommandLine.Usage()
		os.Exit(3)
	}
	if *paramPipelineString == "" {
		fmt.Fprintln(os.Stderr, "Parameter 'pipeline' is empty.")
		flag.CommandLine.Usage()
		os.Exit(4)
	}

	app.org = *paramOrgString
	app.prj = *paramPrjString
	app.token = *paramTokenString
	app.pipeline = *paramPipelineString
	app.branch = *paramBranchString

	for i := 0; i < len(paramsSlice); i++ {
		if strings.Contains(paramsSlice[i], "=") {
			app.parameters = append(app.parameters, paramsSlice[i])
		} else {
			fmt.Fprintln(os.Stderr, "Parameters 'param' does not contain '='.")
		}
	}

	app.infoLog = *paramInfoOutput
	app.warnLog = *paramWarnOutput
	app.verboseLog = *paramVerboseOutput
}

func showUsage() {
	flag.Usage = func() {
		flagSet := flag.CommandLine
		order := []string{"org", "prj", "token", "pipeline", "branch", "param", "w", "i", "v", "h"}

		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])

		for _, name := range order {
			var b strings.Builder
			fflag := flagSet.Lookup(name)
			fmt.Fprintf(&b, "  -%s", fflag.Name) // Two spaces before -; see next two comments.
			name, usage := flag.UnquoteUsage(fflag)
			if len(name) > 0 {
				b.WriteString(" ")
				b.WriteString(name)
			}
			// Boolean flags of one ASCII letter are so common we
			// treat them specially, putting their usage on the same line.
			if b.Len() <= 4 { // space, space, '-', 'x'.
				b.WriteString("\t")
			} else {
				// Four spaces before the tab triggers good alignment
				// for both 4- and 8-space tab stops.
				b.WriteString("\n    \t")
			}
			b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))
			if fflag.DefValue != "" && fflag.DefValue != "false" && fflag.DefValue != "[]" {
				fmt.Fprintf(&b, " (default %q)", fflag.DefValue)
			}

			fmt.Fprint(flag.CommandLine.Output(), b.String(), "\n")
		}
	}
}

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.ErrorLevel)

	app := &App{}
	app.ParseCommandLine()

	if app.warnLog {
		log.SetLevel(log.WarnLevel)
	}
	if app.infoLog {
		log.SetLevel(log.InfoLevel)
	}
	if app.verboseLog {
		log.SetLevel(log.DebugLevel)
	}

	ctx := context.Background()
	client := initClient(ctx, fmt.Sprintf(ADOURL, app.org), app.token)

	pipelineID := app.getPipelineID(client, ctx)
	if pipelineID == -1 {
		log.Fatalf("Pipeline '%s' does not exists!", app.pipeline)
		os.Exit(20)
	}
	runID := app.runPipeline(client, ctx, pipelineID)
	if runID == -1 {
		log.Fatalf("Pipeline '%s' start failed.", app.pipeline)
		os.Exit(21)
	}
	exitCode := app.logStatus(client, ctx, pipelineID, runID)
	if exitCode == 3 {
		log.Warnf("It was not possible to identify the correct return value for pipeline '%s'.", app.pipeline)
	}
	os.Exit(exitCode)
}

func initClient(ctx context.Context, url string, token string) pipelines.Client {
	connection := azuredevops.NewPatConnection(url, token)
	pipelineClient := pipelines.NewClient(ctx, connection)

	return pipelineClient
}

func (app *App) logStatus(client pipelines.Client, ctx context.Context, pipelineId int, runId int) int {
	exitCode := 0
	for {
		result, ec := getRunStatus(client, ctx, app.prj, pipelineId, runId)
		if result == "completed" {
			exitCode = ec
			break
		} else {
			log.Debugf("... '%s (id: %d)' is still running.", app.pipeline, pipelineId)
		}
		time.Sleep(10 * time.Second)
	}
	log.Infof("Pipeline '%s (id: %d)' with run id '%d' finished. Exit code will be %d", app.pipeline, pipelineId, runId, exitCode)

	return exitCode
}

func getRunStatus(client pipelines.Client, ctx context.Context, prj string, pipelineId int, runId int) (string, int) {
	exitCode := 3

	args := &pipelines.GetRunArgs{
		Project:    &prj,
		PipelineId: &pipelineId,
		RunId:      &runId,
	}
	run, err := client.GetRun(ctx, *args)
	if err != nil {
		log.Fatal("Error occurred during get pipeline run status. ", err)
		os.Exit(10)
	}
	if run != nil {
		state := fmt.Sprintf("%v", *run.State)
		if run.FinishedDate != nil {
			finishedDate := (*run.FinishedDate).Time
			runResult := fmt.Sprintf("%v", *run.Result)

			switch runResult {
			case "succeeded":
				exitCode = 0
			case "failed":
				exitCode = 1
			case "canceled":
				exitCode = 2
			default:
				exitCode = 3
			}
			url := *run.Url
			if exitCode == 3 {
				log.Warnf("Pipeline %s is in state '%s' with result '%s', finsihed %s (URL: %s).", *run.Pipeline.Name, state, runResult, finishedDate.Format(time.RFC1123), url)
			} else {
				log.Infof("Pipeline %s is in state '%s' with result '%s', finsihed %s (URL: %s).", *run.Pipeline.Name, state, runResult, finishedDate.Format(time.RFC1123), url)
			}
		}
		return state, exitCode
	}
	return "unknown", exitCode
}

func (app *App) getParameters() map[string]string {
	p := make(map[string]string)

	for _, kvp := range app.parameters {
		kv := strings.Split(kvp, "=")
		if len(kv) == 2 {
			p[kv[0]] = kv[1]
		}
	}

	return p
}

func (app *App) runPipeline(client pipelines.Client, ctx context.Context, pipelineID int) int {
	runId := -1

	m := make(map[string]pipelines.RepositoryResourceParameters)
	m["self"] = pipelines.RepositoryResourceParameters{
		RefName: &app.branch,
	}

	v := make(map[string]string)
	pvars := app.getParameters()
	for key, value := range pvars {
		v[key] = value
	}

	params := &pipelines.RunPipelineParameters{
		Resources: &pipelines.RunResourcesParameters{
			Repositories: &m,
		},
		TemplateParameters: &v,
	}

	args := &pipelines.RunPipelineArgs{
		RunParameters: params,
		Project:       &app.prj,
		PipelineId:    &pipelineID,
	}
	run, err := client.RunPipeline(ctx, *args)
	if err != nil {
		log.Fatal(err)
	}
	if run != nil {
		runId = *run.Id
		runState := fmt.Sprintf("%v", *run.State)
		log.Debugf("Run pipeline '%s'. Run id is '%d' and state is '%s'.", app.pipeline, runId, runState)
	}
	return runId
}

func (app *App) getPipelineID(client pipelines.Client, ctx context.Context) int {
	args := &pipelines.ListPipelinesArgs{
		Project: &app.prj,
	}
	result, err := client.ListPipelines(ctx, *args)
	if err != nil {
		log.Fatal("Error occurred during get pipelines call.", err)
		os.Exit(1)
	}
	i := 0
	pid := -1
	for _, pref := range *result {
		if pid == -1 {
			pid = app.getID(pref)
		}
		if pid != -1 {
			return pid
		}
		i++
	}
	return pid
}

func (app *App) getID(pipeline pipelines.Pipeline) int {
	if fmt.Sprintf("%v", *pipeline.Name) == app.pipeline {
		log.Infof("Pipeline %s has ID %d.", app.pipeline, *pipeline.Id)
		return *pipeline.Id
	} else {
		return -1
	}
}
