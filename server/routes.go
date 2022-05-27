package http

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/OdyseeTeam/e-mage/config"
	"github.com/gin-gonic/gin"
	//"github.com/golang/groupcache/singleflight"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/sirupsen/logrus"
)

//var sf = singleflight.Group{}

type uploadResponse struct {
	Url      string `json:"url"`
	FileName string `json:"file_name"`
	Message  string `json:"message"` //backward compatibility
}

func (s *Server) uploadHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Recovered from panic: %v", r)
		}
	}()
	file, err := c.FormFile("file-input")
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errors.Err(err))
		return
	}
	f, err := file.Open()
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errors.Err(err))
		return
	}
	defer func(f multipart.File) {
		_ = f.Close()
	}(f)

	newImage, err := ioutil.ReadAll(f)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errors.Err(err))
		return
	}

	optimized, originalMime, optimizedMime, err := s.optimizer.Optimize(newImage, 90, 0, 0)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	h := md5.New()
	h.Write(newImage)
	hashedName := hex.EncodeToString(h.Sum(nil))
	c.Header("X-mirage-saved-bytes", fmt.Sprintf("%d", len(newImage)-len(optimized)))
	c.Header("X-mirage-compression-ratio", fmt.Sprintf("%.2f:1", float64(len(newImage))/float64(len(optimized))))
	c.Header("X-mirage-original-mime", originalMime)
	c.Header("X-mirage-optimized-mime", optimizedMime)
	c.JSON(http.StatusOK, uploadResponse{
		FileName: fmt.Sprintf("%s.webp", hashedName),
		Url:      fmt.Sprintf("%s%s.webp", config.CdnUrl, hashedName),
		Message:  fmt.Sprintf("%s%s.webp", config.CdnUrl, hashedName),
	})
}

func (s *Server) recoveryHandler(c *gin.Context, err interface{}) {
	c.JSON(500, gin.H{
		"title": "Error",
		"err":   err,
	})
}

func (s *Server) ErrorHandle(c *gin.Context) {
	c.Next()
	err := c.Errors.Last()
	if err == nil {
		return
	}
	logrus.Errorln(errors.FullTrace(err))
	c.String(-1, err.Error())
}

func (s *Server) addCSPHeaders(c *gin.Context) {
	c.Header("Report-To", `{"group":"default","max_age":31536000,"endpoints":[{"url":"https://6fd448c230d0731192f779791c8e45c3.report-uri.com/a/d/g"}],"include_subdomains":true}`)
	c.Header("Content-Security-Policy", "script-src 'none'; report-uri https://6fd448c230d0731192f779791c8e45c3.report-uri.com/r/d/csp/enforce; report-to default")
}
