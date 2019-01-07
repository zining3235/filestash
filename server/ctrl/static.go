package ctrl

import (
	"crypto/md5"
	"encoding/base32"
	. "github.com/mickael-kerjean/nuage/server/common"
	"io"
	"text/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var ETAGS map[string]string = make(map[string]string)

func StaticHandler(_path string) func(App, http.ResponseWriter, *http.Request) {
	return func(ctx App, res http.ResponseWriter, req *http.Request) {
		var srcPath string
		var err      error

		if srcPath, err = JoinPath(GetAbsolutePath(_path), req.URL.Path); err != nil {
			http.NotFound(res, req)
			return
		}
		if strings.HasPrefix(_path, "/") == true {
			http.NotFound(res, req)
			return
		}

		ServeFile(res, req, srcPath)
	}
}

func IndexHandler(_path string) func(App, http.ResponseWriter, *http.Request) {
	return func(ctx App, res http.ResponseWriter, req *http.Request) {
		if req.URL.String() != URL_SETUP && Config.Get("auth.admin").String() == "" {
			http.Redirect(res, req, URL_SETUP, http.StatusTemporaryRedirect)
			return
		}

		srcPath := GetAbsolutePath(_path)
		ServeFile(res, req, srcPath)
	}
}

func AboutHandler(ctx App, res http.ResponseWriter, req *http.Request) {
	page := `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" name="viewport">
    <style>
      html { background: #f4f4f4; color: #455164; font-size: 16px; font-family: -apple-system,system-ui,BlinkMacSystemFont,Roboto,"Helvetica Neue",Arial,sans-serif; }
      body { text-align: center; padding-top: 50px; text-align: center; }
      h1 { font-weight: 200; }
      p { opacity: 0.7; }
      span { font-size: 0.7em; opacity: 0.7; }
    </style>
  </head>
  <body>
     <h1> {{index .App 0}} <span>({{index .App 1}})</span> </h1>
     <p>{{range .Plugins}}
       {{ index . 0 }} <span>({{ index . 1 }})</span> <br>{{end}}
     </p>
  </body>
</html>`
	t, _ := template.New("about").Parse(page)
	t.Execute(res, struct {
		App     []string
		Plugins [][]string
	}{
		App:     []string{"Nuage " + APP_VERSION, BUILD_NUMBER + "_" + hashFile(filepath.Join(GetCurrentDir(), "/nuage"), 6)},
		Plugins: func () [][]string {
			pPath := filepath.Join(GetCurrentDir(), PLUGIN_PATH)
			file, err := os.Open(pPath)
			if err != nil {
				return [][]string{
					[]string{"N/A", ""},
				}
			}
			files, err := file.Readdir(0)
			if err != nil {
				return [][]string{
					[]string{"N/A", ""},
				}
			}
			plugins := make([][]string, 0)
			plugins = append(plugins, []string {
				"config.json",
				hashFile(filepath.Join(GetCurrentDir(), "/data/config/config.json"), 6),
			})
			for i:=0; i < len(files); i++ {
				plugins = append(plugins, []string{
					files[i].Name(),
					hashFile(pPath + "/" + files[i].Name(), 6),
				})
			}
			return plugins
		}(),
	})
}

func hashFile (path string, n int) string {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return base32.HexEncoding.EncodeToString(h.Sum(nil))[:n]
}

func ServeFile(res http.ResponseWriter, req *http.Request, filePath string) {
	zFilePath := filePath + ".gz"
	if req.Header.Get("If-None-Match") != "" {
		if req.Header.Get("If-None-Match") == ETAGS[filePath] {
			res.WriteHeader(http.StatusNotModified)
			return
		} else if req.Header.Get("If-None-Match") == ETAGS[zFilePath] {
			res.WriteHeader(http.StatusNotModified)
			return
		}
	}
	head := res.Header()

	if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		if file, err := os.OpenFile(zFilePath, os.O_RDONLY, os.ModePerm); err == nil {
			head.Set("Content-Encoding", "gzip")
			if ETAGS[zFilePath] == "" {
				ETAGS[zFilePath] = hashFile(zFilePath, 10)
			}
			head.Set("Etag", ETAGS[zFilePath])
			io.Copy(res, file)
			return
		}
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		http.NotFound(res, req)
		return
	}
	if ETAGS[filePath] == "" {
		ETAGS[filePath] = hashFile(filePath, 10)
	}
	head.Set("Etag", ETAGS[filePath])
	io.Copy(res, file)
}