package main

import (
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"encoding/base64"
    "net"
    "github.com/skip2/go-qrcode"
	"strings"
)


const (
	addr      = ":8080"
	uploadDir = "./data/uploads"
)

type FileItem struct {
	Name    string
	SizeKB  int64
	ModTime string
	URL     string
	DelURL  string
}

var pageTmpl = template.Must(template.New("index").Parse(`
<!doctype html>
<html>
<head>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>DropGo</title>
  <style>
    body { font-family: -apple-system, Arial; padding: 16px; max-width: 720px; margin: 0 auto; }
    h1 { margin: 8px 0 16px; }
    .card { padding: 12px; border: 1px solid #eee; border-radius: 12px; margin: 12px 0; }
    .btn { padding: 10px 14px; border-radius: 10px; border: 0; background: #1677ff; color: #fff; }
    .btn2 { padding: 8px 10px; border-radius: 10px; border: 1px solid #ddd; background: #fff; }
    .row { display:flex; gap: 8px; align-items:center; flex-wrap: wrap; }
    input[type=file]{ max-width: 100%; }
    a { color:#1677ff; text-decoration:none; }
    small { color:#666; }
  </style>
</head>
<body>
  <h1>DropGo</h1>
  <div class="card">
  <h3 style="margin-top:0">Вход с мобильного устройства</h3>
  <p><a href="{{.URL}}">{{.URL}}</a></p>
  <img src="/qr.png" width="220" style="border-radius:12px; border:1px solid #eee;">
  <p><small>Сканируй камерой iPhone/Android и сразу открывай сайт!</small></p>
  </div>
  <div class="card">
    <form enctype="multipart/form-data" action="/upload" method="post">
      <div class="row">
        <input type="file" name="file" required />
        <button class="btn" type="submit">Загрузить</button>
      </div>
    </form>
    {{if .Msg}}<p><small>{{.Msg}}</small></p>{{end}}
  </div>
<div class="row" style="margin-top:10px">
  <a class="btn2" href="/gallery">Открыть галерею</a>
</div>
  <div class="card">
    <h3 style="margin-top:0">Файлы ({{len .Files}})</h3>
    {{if not .Files}}
      <p><small>Пока пусто. Закинь фотку с айфона 🙂</small></p>
    {{else}}
      <ul>
        {{range .Files}}
          <li style="margin: 10px 0">
            <div class="row">
              <a href="{{.URL}}">{{.Name}}</a>
              <small>({{.SizeKB}} KB, {{.ModTime}})</small>
              <form action="{{.DelURL}}" method="post" style="display:inline">
                <button class="btn2" type="submit">Удалить</button>
              </form>
            </div>
          </li>
        {{end}}
      </ul>
    {{end}}
  </div>

</body>
</html>
`))
var galleryTmpl = template.Must(template.New("gallery").Parse(`
<!doctype html>
<html>
<head>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>DropGo — Галерея</title>
  <style>
    body { font-family: -apple-system, Arial; padding: 16px; max-width: 900px; margin: 0 auto; }
    a { color:#1677ff; text-decoration:none; }
    .top { display:flex; justify-content:space-between; align-items:center; gap:12px; flex-wrap:wrap; }
    .grid { display:grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px; margin-top: 16px; }
    .item { border:1px solid #eee; border-radius: 12px; overflow:hidden; background:#fff; }
    .item img { width:100%; height:140px; object-fit:cover; display:block; }
    .cap { padding:8px; font-size:12px; color:#333; word-break: break-all; }
  </style>
</head>
<body>
  <div class="top">
    <h2 style="margin:0">Галерея</h2>
    <a href="/">← назад</a>
  </div>

  <div class="grid">
    {{range .Images}}
      <div class="item">
        <a href="{{.URL}}">
          <img src="{{.URL}}" alt="{{.Name}}">
        </a>
        <div class="cap">{{.Name}}</div>
      </div>
    {{end}}
  </div>

  {{if not .Images}}
    <p>Пока нет картинок.</p>
  {{end}}
</body>
</html>
`))

func isImage(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	entries, _ := os.ReadDir(uploadDir)
	images := make([]FileItem, 0)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isImage(e.Name()) {
			continue
		}
		name := filepath.Base(e.Name())
		images = append(images, FileItem{
			Name: name,
			URL:  "/f/" + name,
		})
	}

	_ = galleryTmpl.Execute(w, struct {
		Images []FileItem
	}{
		Images: images,
	})
}

func safeBaseName(name string) string {
	// оставляем только имя файла, без путей
	name = filepath.Base(name)
	// минимальная защита от странных символов в HTML (для вывода)
	return html.EscapeString(name)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil {
			continue
		}
		// обычно локалка 192.168.x.x / 10.x.x.x / 172.16-31.x.x
		s := ip.String()
		if strings.HasPrefix(s, "192.168.") || strings.HasPrefix(s, "10.") || strings.HasPrefix(s, "172.") {
			return s
		}
	}
	return "127.0.0.1"
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	host := getLocalIP()
    url := "http://" + host + addr

    png, _ := qrcode.Encode(url, qrcode.Medium, 256)
    qrB64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	msg := r.URL.Query().Get("msg")

	entries, _ := os.ReadDir(uploadDir)
	files := make([]FileItem, 0, len(entries))
    
	
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := safeBaseName(e.Name())
		files = append(files, FileItem{
			Name:    name,
			SizeKB: info.Size() / 1024,
			ModTime: info.ModTime().Format("02.01 15:04"),
			URL:     "/f/" + e.Name(),
			DelURL:  "/delete?name=" + e.Name(),
		})
	}

	_ = pageTmpl.Execute(w, struct {
	Msg   string
	Files []FileItem
	URL   string
	QR    string
}{
	Msg:   msg,
	Files: files,
	URL:   url,
	QR:    qrB64,
})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// лимит на размер (например 300MB) чтобы случайно не уложить комп
	r.Body = http.MaxBytesReader(w, r.Body, 300<<20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, "/?msg=Ошибка+загрузки", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// делаем имя уникальным, чтобы не затирать файлы
	base := filepath.Base(handler.Filename)
	ext := filepath.Ext(base)
	nameOnly := base[:len(base)-len(ext)]
	newName := fmt.Sprintf("%s_%d%s", nameOnly, time.Now().Unix(), ext)

	dstPath := filepath.Join(uploadDir, newName)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Redirect(w, r, "/?msg=Ошибка+сохранения", http.StatusSeeOther)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Redirect(w, r, "/?msg=Ошибка+копирования", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?msg=Загружено:+%s", http.StatusSeeOther)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	// /f/<name>
	name := filepath.Base(r.URL.Path[len("/f/"):])
	http.ServeFile(w, r, filepath.Join(uploadDir, name))
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	name := filepath.Base(r.URL.Query().Get("name"))
	if name != "" {
		_ = os.Remove(filepath.Join(uploadDir, name))
	}
	http.Redirect(w, r, "/?msg=Удалено", http.StatusSeeOther)
}
func qrHandler(w http.ResponseWriter, r *http.Request) {
	host := getLocalIP()
	url := "http://" + host + addr

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}
func main() {
	_ = os.MkdirAll(uploadDir, os.ModePerm)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/f/", fileHandler)
	http.HandleFunc("/delete", deleteHandler)
    http.HandleFunc("/qr.png", qrHandler)
	http.HandleFunc("/gallery", galleryHandler)
	fmt.Println("DropGo started on", addr)
	_ = http.ListenAndServe(addr, nil)
}