package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
)

func getModelName(w http.ResponseWriter, r *http.Request) string {
	modelQueryValues := r.URL.Query()["model"]
	modelName := ""
	if len(modelQueryValues) > 0 {
		modelName = modelQueryValues[0]
	}

	if modelName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Please set the 'model' name as a query parameter."))
		return ""
	}
	return modelName
}

func checkApiKey(r *http.Request) bool {
	apiKey := os.Getenv("API_KEY")
	return r.Header.Get("Authorization") == apiKey && apiKey != ""
}

func uploadFile(w http.ResponseWriter, r *http.Request, mm *ModelManager) {
	if !checkApiKey(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	modelName := getModelName(w, r)
	if modelName == "" {
		return
	}

	fmt.Println("Reciving model: " + modelName)

	tempFile, tempFilePath, err := mm.createTempFile()
	if err != nil {
		fmt.Println("Failed to create tempFile: " + err.Error())
		w.Write([]byte("Failed to create the temporary file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	reader, err := r.MultipartReader()
	if err != nil {
		w.Write([]byte("Please upload the model as a multipart/form-data."))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	part, err := reader.NextPart()
	if part.FormName() != "model" || err == io.EOF {
		w.Write([]byte("Please upload one part with the name 'model'"))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err, part.FormName())
		return
	}
	if err != nil {
		w.Write([]byte("Error while getting 'model' field"))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
		return
	}

	_, err = io.Copy(tempFile, part)
	if err != nil {
		fmt.Println("Failed to copy: stream to file")
		w.Write([]byte("Could not read stream / write file"))
		w.WriteHeader(http.StatusInternalServerError)
	}

	fmt.Println("Upload completed: " + part.FileName())
	mm.recievedNewModel(tempFilePath, modelName)
}

func downloadFile(w http.ResponseWriter, r *http.Request, mm *ModelManager) {
	modelName := getModelName(w, r)
	if modelName == "" {
		return
	}

	modelHash := mm.knownModels[modelName]
	if modelHash == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ifNoneMatch := r.Header["If-None-Match"]
	if ifNoneMatch != nil {
		if slices.Contains(ifNoneMatch, modelHash) {
			w.WriteHeader(304)
			return
		}
	}

	fmt.Println("Serving model: " + modelName)
	w.Header().Add("ETag", modelHash)
	http.ServeFile(w, r, mm.getModelPath(modelName))
}

func main() {
	modelManager, err := initModelManager("./rasaModels")
	if err != nil {
		fmt.Println("Could not create directory for models.")
		os.Exit(1)
	}

	uploadHandler := func(w http.ResponseWriter, r *http.Request) {
		uploadFile(w, r, modelManager)
	}

	downloadHandler := func(w http.ResponseWriter, r *http.Request) {
		downloadFile(w, r, modelManager)
	}

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/download", downloadHandler)
	fmt.Println("Listing on :8080")
	http.ListenAndServe(":8080", nil)
}
