package admin

import (
	"context"
	"io"
	"mime"
	"path/filepath"
	"strings"

	flows "github.com/example/medusa-go-clone/internal/coreflows/file/workflows"
	"github.com/example/medusa-go-clone/internal/modules/file/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/gin-gonic/gin"
)

type FileService interface {
	Upload(context.Context, []models.UploadInput) ([]models.File, error)
	Delete(context.Context, []string) error
	Get(context.Context, string) (models.File, error)
	PresignedUpload(context.Context, models.PresignedInput) (models.PresignedOutput, error)
}
type FileHTTP struct{ Service FileService }

func (h *FileHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	a := router.Group("/admin", auth)
	a.POST("/uploads", h.upload)
	a.POST("/uploads/presigned-urls", h.presigned)
	a.GET("/uploads/:id", h.get)
	a.DELETE("/uploads/:id", h.delete)
}
func (h *FileHTTP) upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "No files were uploaded"})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "No files were uploaded"})
		return
	}
	input := make([]models.UploadInput, 0, len(files))
	for _, file := range files {
		f, e := file.Open()
		if e != nil {
			writeModuleError(c, e)
			return
		}
		content, e := io.ReadAll(f)
		_ = f.Close()
		if e != nil {
			writeModuleError(c, e)
			return
		}
		input = append(input, models.UploadInput{Filename: file.Filename, MIMEType: file.Header.Get("Content-Type"), Content: content, Access: "public"})
	}
	out, err := flows.UploadFiles(c, h.Service, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"files": out})
}
func (h *FileHTTP) presigned(c *gin.Context) {
	var in models.PresignedInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	media, _, err := mime.ParseMediaType(in.MIMEType)
	if err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "Invalid file type \"" + in.MIMEType + "\""})
		return
	}
	_, sub, ok := strings.Cut(media, "/")
	if !ok || sub == "" {
		c.JSON(400, gin.H{"type": "invalid_data", "message": "Invalid file type"})
		return
	}
	ext := sub
	filename := entityid.New("") + "." + ext
	if in.Access == "" {
		in.Access = "private"
	}
	in.Filename = filename
	out, err := h.Service.PresignedUpload(c, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	out.Extension = ext
	c.JSON(200, out)
}
func (h *FileHTTP) get(c *gin.Context) {
	v, err := h.Service.Get(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"file": v})
}
func (h *FileHTTP) delete(c *gin.Context) {
	id := filepath.ToSlash(c.Param("id"))
	if err := flows.DeleteFiles(c, h.Service, []string{id}); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "object": "file", "deleted": true})
}
