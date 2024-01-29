package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/yaml"
)

func findFiles(root string, ext string) []string {
	var a []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, s)
		}
		return nil
	})
	return a
}

func verifyPipeline(path string) error {
	var err error
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix(fmt.Sprintf("%s ", path))

	yamlFile, readErr := os.ReadFile(path)
	if readErr != nil {
		return readErr
	}

	var pipeline v1.Pipeline
	unmarshErr := yaml.Unmarshal(yamlFile, &pipeline)
	if unmarshErr != nil {
		return readErr
	}
	if pipeline.Kind != "Pipeline" {
		return errors.New("Not a valid pipeline")
	}

	paramErr := pipeline.Spec.Params.ValidateNoDuplicateNames()
	if paramErr != nil {
		return errors.New(paramErr.Error())
	}

	return err
}

func verifyTask(path string) error {
	var err error
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix(fmt.Sprintf("%s ", path))

	yamlFile, readErr := os.ReadFile(path)
	if readErr != nil {
		return readErr
	}

	var task v1.Task
	unmarshErr := yaml.Unmarshal(yamlFile, &task)
	if unmarshErr != nil {
		return readErr
	}
	if task.Kind != "Task" {
		return errors.New("Not a valid task")
	}

	paramErr := task.Spec.Params.ValidateNoDuplicateNames()
	if paramErr != nil {
		return errors.New(paramErr.Error())
	}

	return err
}

func main() {
	pipelinePathFlag := flag.String("pipeline-path", "pipeline", "path to pipeline directory")
	taskPathFlag := flag.String("task-path", "task", "path to task directory")
	flag.Parse()

	var failedFiles []string

	log.Printf("Checking pipelines...")
	for _, s := range findFiles(*pipelinePathFlag, ".yaml") {
		log.SetPrefix("")
		log.Printf("Verifying %s\n", s)
		err := verifyPipeline(s)
		if err != nil {
			log.Println(err)
			failedFiles = append(failedFiles, s)
		}
		log.SetPrefix("")
	}

	log.Printf("Checking tasks...")
	for _, s := range findFiles(*taskPathFlag, ".yaml") {
		log.SetPrefix("")
		log.Printf("Verifying %s\n", s)
		err := verifyTask(s)
		if err != nil {
			log.Println(err)
			failedFiles = append(failedFiles, s)
		}
		log.SetPrefix("")
	}

	if len(failedFiles) > 0 {
		log.Fatalf("Found %d incorrect files %s", len(failedFiles), failedFiles)
	}
	fmt.Println("All good!")
}
