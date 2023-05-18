package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelinerun"
	"github.com/spf13/pflag"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

var (
	configDefault string
	configLocal   string
	verbose       bool
)

func init() {
	flag.StringVar(&configDefault, "config-default", "", "The path to the default trigger config.")
	flag.StringVar(&configLocal, "config-local", "", "The path to the local trigger config `.tekton.yaml`.")
	flag.BoolVar(&verbose, "verbose", false, "Print generated pipelineRuns.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

func readAndUnmarshalConfig(path string) (*pipelineconfig.Config, error) {
	conf, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read default config: %v", err)
	}

	cfg := &pipelineconfig.Config{}
	err = yaml.Unmarshal(conf, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return cfg, nil
}

func toPipelineRun(p ...*pipelinerun.PipelineRun) (*v1beta1.PipelineRun, error) {
	tr := &pipelinerun.PipelineRun{}
	err := tr.MergeAll(p...)
	if err != nil {
		return nil, err
	}
	return tr.PipelineRun()
}

func PipelineRuns(c pipelineconfig.Config) ([]*v1beta1.PipelineRun, error) {
	var prs []*v1beta1.PipelineRun
	for _, t := range c.Triggers {
		for _, p := range t.Pipelines {
			pr, err := toPipelineRun(&c.Defaults, &t.Defaults, &p)
			if err != nil {
				return nil, err
			}
			prs = append(prs, pr)
		}
	}
	return prs, nil
}

func processPipelineRuns(cfg *pipelineconfig.Config) {
	pprs, err := PipelineRuns(*cfg)
	if err != nil {
		log.Fatalf("Can't get PPRs: %v", err)
	}

	for _, p := range pprs {
		pp, err := yaml.Marshal(p)
		if err != nil {
			log.Fatalf("Can't marshal ppr: %v", err)
		}
		if verbose {
			fmt.Println(string(pp))
		}
	}
}

func main() {
	if configDefault == "" || configLocal == "" {
		log.Fatal("Please provide both config-default and config-local")
	}
	ppDefConf, err := readAndUnmarshalConfig(configDefault)
	if err != nil {
		log.Fatal(err)
	}
	ppRepConf, err := readAndUnmarshalConfig(configLocal)
	if err != nil {
		log.Fatal(err)
	}

	err = ppDefConf.Merge(ppRepConf)
	if err != nil {
		log.Fatalf("Can't merge configs: %v", err)
	}

	processPipelineRuns(ppDefConf)
}
