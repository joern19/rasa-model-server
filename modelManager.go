package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

type ModelManager struct {
	workDir     string
	modelDir    string
	knownModels map[string]string
}

func initModelManager(root string) (*ModelManager, error) {
	workDir := path.Clean(root)
	modelDir := path.Join(workDir, "models")

	err := os.MkdirAll(modelDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	knownModels := map[string]string{}

	items, _ := os.ReadDir(modelDir)
	for _, item := range items {
		if item.Type().IsRegular() {
			name := item.Name()
			hash, _ := calculateHash(path.Join(modelDir, name))
			knownModels[name] = base64.StdEncoding.EncodeToString(hash)
		}
	}

	return &ModelManager{
		workDir:     workDir,
		modelDir:    modelDir,
		knownModels: knownModels,
	}, nil
}

func calculateHash(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return h.Sum(nil), nil
}

func (mm *ModelManager) recievedNewModel(modelPath string, modelName string) {
	newModelPath := path.Join(mm.modelDir, modelName)
	err := os.Rename(modelPath, newModelPath)
	if err != nil {
		fmt.Println("Failed to move model into modelFolder")
		fmt.Println(modelPath + " -> " + newModelPath)
		fmt.Println(err.Error())
		os.Exit(1)
		return
	}
	hash, err := calculateHash(newModelPath)
	if err != nil {
		fmt.Println("Failed to calculate hash for: " + newModelPath)
		os.Exit(1)
		return
	}

	mm.knownModels[modelName] = base64.StdEncoding.EncodeToString(hash)
}

func (mm *ModelManager) createTempFile() (*os.File, string, error) {
	tempFile, err := os.CreateTemp(mm.workDir, "rasaModelUpload-*")
	modelFileInfo, _ := tempFile.Stat()
	return tempFile, path.Join(mm.workDir, modelFileInfo.Name()), err
}

func (mm *ModelManager) getModelPath(modelName string) string {
	return path.Join(mm.modelDir, modelName)
}
