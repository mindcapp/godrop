package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Ошибка загрузки", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dst, err := os.Create("./uploads/" + handler.Filename)
	if err != nil {
		http.Error(w, "Ошибка сохранения", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	io.Copy(dst, file)

	w.Write([]byte("Файл успешно загружен"))
}

func main() {
	os.MkdirAll("./uploads", os.ModePerm)

	http.HandleFunc("/upload", uploadHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<h1>DropGo</h1>
			<form enctype="multipart/form-data" action="/upload" method="post">
				<input type="file" name="file">
				<input type="submit" value="Загрузить">
			</form>
		`))
	})

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}