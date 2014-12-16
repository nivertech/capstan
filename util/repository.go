/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"errors"
	"fmt"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/image"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type Repo struct {
	Path string
}

func NewRepo() *Repo {
	root := os.Getenv("CAPSTAN_ROOT")
	if root == "" {
		root = filepath.Join(HomePath(), "/.capstan/repository/")
	}
	return &Repo{
		Path: root,
	}
}

type ImageInfo struct {
	FormatVersion string `yaml:"format_version"`
	Version       string
	Created       string
	Description   string
	Build         string
}

func (r *Repo) ImportImage(imageName string, file string, version string, created string, description string, build string) error {
	format, err := image.Probe(file)
	if err != nil {
		return err
	}
	var hypervisor string
	switch format {
	case image.VDI:
		hypervisor = "vbox"
	case image.QCOW2:
		hypervisor = "qemu"
	case image.VMDK:
		hypervisor = "vmware"
	default:
		return fmt.Errorf("%s: unsupported image format", file)
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such file", file))
	}
	fmt.Printf("Importing %s...\n", imageName)
	dir := filepath.Dir(r.ImagePath(hypervisor, imageName))
	err = os.MkdirAll(dir, 0775)
	if err != nil {
		return errors.New(fmt.Sprintf("%s: mkdir failed", dir))
	}

	dst := r.ImagePath(hypervisor, imageName)
	cmd := CopyFile(file, dst)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	info := ImageInfo{
		FormatVersion: "1",
		Version: version,
		Created: created,
		Description: description,
		Build: build,
	}
	value, err := yaml.Marshal(info)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "index.yaml"), value, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ImageExists(hypervisor, image string) bool {
	file := r.ImagePath(hypervisor, image)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func (r *Repo) RemoveImage(image string) error {
	path := filepath.Join(r.Path, image)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such image\n", image))
	}
	fmt.Printf("Removing %s...\n", image)
	cmd := exec.Command("rm", "-rf", path)
	_, err := cmd.Output()
	return err
}

func (r *Repo) ImagePath(hypervisor string, image string) string {
	return filepath.Join(r.Path, image, fmt.Sprintf("%s.%s", filepath.Base(image), hypervisor))
}

func (r *Repo) ListImages() {
	fmt.Println(FileInfoHeader())
	namespaces, _ := ioutil.ReadDir(r.Path)
	for _, n := range namespaces {
		images, _ := ioutil.ReadDir(filepath.Join(r.Path, n.Name()))
		nrImages := 0
		nrFiles := 0
		for _, i := range images {
			if i.IsDir() {
				info := MakeFileInfo(r.Path, n.Name(), i.Name())
				if info == nil {
					fmt.Println(n.Name() + "/" + i.Name())
				} else {
					fmt.Println(info.String())
				}
				nrImages++
			} else {
				nrFiles++
			}
		}
		// Image is directly at repository root with no namespace:
		if nrImages == 0 && nrFiles != 0 {
			fmt.Println(n.Name())
		}
	}
}

func (r *Repo) DefaultImage() string {
	if !core.IsTemplateFile("Capstanfile") {
		return ""
	}
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	image := path.Base(pwd)
	return image
}
