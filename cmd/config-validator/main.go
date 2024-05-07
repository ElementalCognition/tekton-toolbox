package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelinerun"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/yaml"
)

var (
	configDefault    string
	configLocal      string
	verbose          bool
	alphaFeatureGate bool
)

func init() {
	color.NoColor = false
	flag.StringVar(&configDefault, "config-default", "", "The path to the default trigger config.")
	flag.StringVar(&configLocal, "config-local", "", "The path to the local trigger config `.tekton.yaml`.")
	flag.BoolVar(&verbose, "verbose", false, "Print generated pipelineRuns.")
	flag.BoolVar(&alphaFeatureGate, "alphaFeatureGate", false, "Config enable-api-fields alpha.")
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

func toPipelineRun(p ...*pipelinerun.PipelineRun) (*v1.PipelineRun, error) {
	tr := &pipelinerun.PipelineRun{}
	err := tr.MergeAll(p...)
	if err != nil {
		return nil, err
	}
	return tr.PipelineRun()
}

func PipelineRuns(c pipelineconfig.Config) ([]*v1.PipelineRun, error) {
	var prs []*v1.PipelineRun
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

func processPipelineRuns(cfg *pipelineconfig.Config) ([]byte, bool) {
	pprs, err := PipelineRuns(*cfg)
	if err != nil {
		log.Fatalf("Can't get PPRs: %v", err)
	}

	red := color.New(color.FgHiRed).SprintFunc()
	green := color.New(color.FgHiGreen).SprintFunc()

	var pprYaml []byte
	var allValid = true

	for _, p := range pprs {
		pp, err := yaml.Marshal(p)
		if err != nil {
			log.Fatalf("Can't marshal ppr: %v", red(err))
		}
		pprYaml = append(pprYaml, pp...)

		ctx := context.TODO()
		if alphaFeatureGate {
			ctx = enableAlphaAPIFields(ctx)
		}

		errs := p.Spec.Validate(ctx)
		if len(errs.Error()) > 0 {
			log.Printf("Pipeline %s: %v. Error: %v", p.GenerateName, red("Failed"), red(errs))
			allValid = false
		} else {
			log.Printf("Pipeline %s: %v", p.GenerateName, green("Valid"))
		}
	}
	return pprYaml, allValid
}

func enableAlphaAPIFields(ctx context.Context) context.Context {
	featureFlags, _ := config.NewFeatureFlagsFromMap(map[string]string{
		"enable-api-fields": config.AlphaAPIFields,
	})
	cfg := &config.Config{
		Defaults: &config.Defaults{
			DefaultTimeoutMinutes: config.DefaultTimeoutMinutes,
		},
		FeatureFlags: featureFlags,
	}
	return config.ToContext(ctx, cfg)
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

	ppr, fail := processPipelineRuns(ppDefConf)
	if verbose {
		fmt.Printf("---\n%s", string(ppr))
	}
	if !fail {
		log.Fatal("Pipelines validation failed.")
	}
}
