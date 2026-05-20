package docker

import (
"archive/tar"
"fmt"
"io"
"os"
"path/filepath"
"strings"
)

type RunConfig struct {
ProjectID     string
ImageTag      string
Domain        string   // primary domain
ExtraDomains  []string // additional custom domains
ContainerPort int
EnvVars       []string
CPULimit      float64
MemoryLimitMB int64
}

func TagForProject(projectID string) string {
return fmt.Sprintf("ld_img_%s:latest", strings.ToLower(projectID))
}

func CreateBuildArchive(srcDir string) (io.ReadCloser, error) {
pr, pw := io.Pipe()
go func() {
if err := tarDir(srcDir, pw); err != nil {
pw.CloseWithError(err)
} else {
pw.Close()
}
}()
return pr, nil
}

func tarDir(srcDir string, w io.Writer) error {
tw := tar.NewWriter(w)
defer tw.Close()
return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
if err != nil {
return err
}
rel, err := filepath.Rel(srcDir, path)
if err != nil {
return err
}
rel = filepath.ToSlash(rel)
hdr := &tar.Header{
Name:    rel,
Size:    info.Size(),
Mode:    int64(info.Mode()),
ModTime: info.ModTime(),
}
if info.IsDir() {
hdr.Typeflag = tar.TypeDir
hdr.Name += "/"
return tw.WriteHeader(hdr)
}
hdr.Typeflag = tar.TypeReg
if err := tw.WriteHeader(hdr); err != nil {
return err
}
f, err := os.Open(path)
if err != nil {
return err
}
defer f.Close()
_, err = io.Copy(tw, f)
return err
})
}
