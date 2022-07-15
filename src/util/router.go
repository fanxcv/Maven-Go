package util

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	Engine     *gin.Engine
	fs         http.FileSystem
	fileServer http.Handler
	client     = resty.New()
)

func init() {
	fs = http.Dir(config.LocalRepository)
	fileServer = http.StripPrefix(path.Join("/", config.Context), http.FileServer(fs))

	Engine = gin.Default()
	Engine.PUT("/:context/:libName/*filePath", put)
	Engine.GET("/:context/:libName/*filePath", get)
	Engine.HEAD("/:context/:libName/*filePath", get)
}

func get(c *gin.Context) {
	mode, repository, err := checkAndGet(c)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	if mode != 4 && mode != 6 {
		c.String(http.StatusForbidden, "repository not support read")
		return
	}

	filePath := c.Param("filePath")
	ext := path.Ext(filePath)
	if ext == "" && !strings.HasSuffix(filePath, "/") {
		c.Redirect(http.StatusMovedPermanently, c.Request.RequestURI+"/")
		return
	}
	localFilePath := path.Join(repository.Id, filePath)

	f, err := fs.Open(localFilePath)
	defer closeFile(f)
	if err != nil && len(repository.Mirror) > 0 {
		// 尝试从url镜像获取返回
		response := readRemote(repository, filePath)
		if response == nil {
			c.String(http.StatusNotFound, "not found")
		}

		c.Data(response.StatusCode(), response.Header().Get("Content-Type"), response.Body())
		return
	}

	if generate := c.Query("generate_md5_sha1"); strings.EqualFold(generate, "true") {
		if !checkAuth(c) {
			c.String(http.StatusUnauthorized, "Unauthorised")
			return
		}

		if err = generateHash(localFilePath); err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("generate hash failed, message: %v\n", err))
		}
	}

	fileServer.ServeHTTP(c.Writer, c.Request)
}

func put(c *gin.Context) {
	if !checkAuth(c) {
		c.String(http.StatusUnauthorized, "Unauthorised")
		return
	}

	length, err1 := strconv.Atoi(c.GetHeader("Content-Length"))
	data, err2 := ioutil.ReadAll(c.Request.Body)
	if err1 != nil || err2 != nil || length <= 0 || length != len(data) {
		log.Errorf("data read failed\n%v\n%v\n", err1, err2)
		c.String(http.StatusInternalServerError, "data read failed")
		return
	}

	mode, repository, err := checkAndGet(c)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	if mode != 6 && mode != 2 {
		c.String(http.StatusForbidden, "repository not support write")
		return
	}

	filePath := c.Param("filePath")
	filePath = path.Join(config.LocalRepository, repository.Id, filePath)
	dirPath := path.Dir(filePath)

	if stat, err := os.Stat(dirPath); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dirPath, 0755); err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("create dir failed. message: %v\n", err))
			return
		}
	} else if !stat.IsDir() {
		c.String(http.StatusInternalServerError, fmt.Sprintf("%s is not a dir\n", dirPath))
		return
	}

	if err = ioutil.WriteFile(filePath, data, 0755); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("write file failed. message: %v\n", err))
		return
	}

	if generate := c.Query("generate_md5_sha1"); strings.EqualFold(generate, "true") {
		if err = generateHash(filePath); err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("generate hash failed, message: %v\n", err))
		}
	}

	c.String(http.StatusOK, "OK")
}

func readRemote(repository *Repository, filePath string) *resty.Response {
	for _, u := range repository.Mirror {
		u = fmt.Sprintf("%s%s", u, filePath)
		if response, err := client.R().Get(u); err != nil {
			log.Errorf("request mirror url '%s' failed", u)
		} else {
			return response
		}
	}
	return nil
}

func generateHash(file string) error {
	stat, err := os.Stat(file)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		dir, err := ioutil.ReadDir(file)
		if err != nil {
			return err
		}
		for _, info := range dir {
			if err = generateHash(info.Name()); err != nil {
				return err
			}
		}
	}
	ext := path.Ext(file)
	if ext != ".xml" && ext != ".jar" && ext != ".pom" {
		return nil
	}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err = touchFile(file, "md5", bytes); err != nil {
		return err
	}
	if err = touchFile(file, "sha1", bytes); err != nil {
		return err
	}
	if err = touchFile(file, "sha256", bytes); err != nil {
		return err
	}
	if err = touchFile(file, "sha512", bytes); err != nil {
		return err
	}
	return nil
}

func touchFile(file string, hash string, bytes []byte) error {
	hashFile := fmt.Sprintf("%s.%s", file, hash)
	if exist, err := checkFileExist(hashFile); err != nil {
		return err
	} else if !exist {
		if err = ioutil.WriteFile(hashFile, getHash(bytes, hash), 0755); err != nil {
			return err
		}
	}
	return nil
}

func getHash(file []byte, hash string) []byte {
	switch hash {
	case "md5":
		return []byte(fmt.Sprintf("%x", md5.Sum(file)))
	case "sha1":
		return []byte(fmt.Sprintf("%x", sha1.Sum(file)))
	case "sha256":
		return []byte(fmt.Sprintf("%x", sha256.Sum256(file)))
	case "sha512":
		return []byte(fmt.Sprintf("%x", sha512.Sum512(file)))
	default:
		return nil
	}
}

func checkFileExist(file string) (bool, error) {
	if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func closeFile(f http.File) {
	if f != nil {
		_ = f.Close()
	}
}

func checkAndGet(c *gin.Context) (mode int, repository *Repository, err error) {
	context := c.Param("context")
	libName := c.Param("libName")
	filePath := c.Param("filePath")
	fullPath := fmt.Sprintf("/%s/%s%s", context, libName, filePath)
	if context != config.Context {
		return 0, nil, errors.New(fmt.Sprintf("not found, url = %s", fullPath))
	}

	// 获取存储库配置
	repository = config.RepositoryStore[libName]
	if repository == nil || repository.Mode == 0 {
		return 0, nil, errors.New(fmt.Sprintf("repository %s is not actived", libName))
	}

	// 判断是否需要转为镜像库
	var mirror *Repository
	if repository.Target != "" {
		if mirror = config.RepositoryStore[repository.Target]; mirror == nil {
			return 0, nil, errors.New(fmt.Sprintf("target repository %s is not found", repository.Target))
		}
	} else {
		mirror = repository
	}

	return repository.Mode, mirror, nil
}

func checkAuth(c *gin.Context) bool {
	authorization := c.GetHeader("Authorization")
	if !strings.HasPrefix(authorization, "Basic ") {
		return false
	}
	// 校验用户
	authorization = strings.TrimSpace(authorization[6:])
	if config.Auth[authorization] == nil {
		return false
	}
	return true
}
