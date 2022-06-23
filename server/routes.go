package http

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/OdyseeTeam/e-mage/config"
	"github.com/OdyseeTeam/e-mage/internal/metrics"
	"github.com/OdyseeTeam/gody-cdn/store"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/sirupsen/logrus"
)

type uploadResponse struct {
	Url      string `json:"url"`
	FileName string `json:"file_name"`
	Type     string `json:"type"`    //backward compatibility
	Message  string `json:"message"` //backward compatibility
}

func (s *Server) getImageHandler(c *gin.Context) {
	resource := c.Param("resource")
	resource = strings.Split(resource, ".")[0]
	imagePtr, err, shared := s.sf.Do(resource, func() (interface{}, error) {
		logrus.Warningf("missed %s", resource)
		image, _, err := s.cache.Get(resource, nil)
		if err != nil {
			return nil, err
		}
		return &image, nil
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrObjectNotFound) {
			status = http.StatusNotFound
		}
		_ = c.AbortWithError(status, err)
		return
	}
	image := *(imagePtr.(*[]byte))
	logrus.Infof("shared: %t", shared)
	contentType := mimetype.Detect(image).String()
	c.Data(200, contentType, image)
}

func (s *Server) uploadHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Recovered from panic: %v", r)
		}
	}()
	metrics.UploadsRunning.Inc()
	defer metrics.UploadsRunning.Dec()

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
	metrics.RequestCount.Inc()

	h := md5.New()
	h.Write(newImage)
	hashedName := hex.EncodeToString(h.Sum(nil))

	alreadyStored, err := s.cache.Has(hashedName, nil)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !alreadyStored {
		err = s.cache.Put(hashedName, optimized, nil)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	c.Header("X-e-mage-saved-bytes", fmt.Sprintf("%d", len(newImage)-len(optimized)))
	c.Header("X-e-mage-compression-ratio", fmt.Sprintf("%.2f:1", float64(len(newImage))/float64(len(optimized))))
	c.Header("X-e-mage-original-mime", originalMime)
	c.Header("X-e-mage-optimized-mime", optimizedMime)
	c.JSON(http.StatusOK, uploadResponse{
		FileName: fmt.Sprintf("%s.webp", hashedName),
		Url:      fmt.Sprintf("%s%s.webp", config.CdnUrl, hashedName),
		Message:  fmt.Sprintf("%s%s.webp", config.CdnUrl, hashedName),
		Type:     "success",
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
	logrus.Errorln(err)
	c.String(-1, err.Error())
}

func (s *Server) addCSPHeaders(c *gin.Context) {
	c.Header("Report-To", `{"group":"default","max_age":31536000,"endpoints":[{"url":"https://6fd448c230d0731192f779791c8e45c3.report-uri.com/a/d/g"}],"include_subdomains":true}`)
	c.Header("Content-Security-Policy", "script-src 'none'; report-uri https://6fd448c230d0731192f779791c8e45c3.report-uri.com/r/d/csp/enforce; report-to default")
}
