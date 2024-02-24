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

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("model")
	if err != nil {
		w.Write([]byte("Could not retrieve the file. Please upload it as model."))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, tempFilePath, err := mm.createTempFile()
	if err != nil {
		fmt.Println("Failed to create tempFile: " + err.Error())
		w.Write([]byte("Failed to create the temporary file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	buffer := make([]byte, 1024*1000)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Failed to read stream")
			w.Write([]byte("Could not read file"))
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, err = tempFile.Write(buffer[:n])
		if err != nil {
			fmt.Println("Failed to write to file")
			w.Write([]byte("Could not write file"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	// return that we have successfully uploaded our file!
	fmt.Fprintf(w, "Successfully Uploaded File\n")
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
