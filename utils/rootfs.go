package utils

import (
	"fmt"
	"path"
)

const (
	ImagePath         = "/var/lib/mydocker/image/"
	RootPath          = "/var/lib/mydocker/overlay2/"
	lowerPathFormat   = RootPath + "%s/lower/"
	upperPathFormat   = RootPath + "%s/upper/"
	workPathFormat    = RootPath + "%s/work/"
	mergedPathFormat  = RootPath + "%s/merged/"
	mountOptionFormat = "lowerdir=%s,upperdir=%s,workdir=%s"
)

func GetRootPath(containerId string) string {
	return path.Join(RootPath, containerId)
}

func GetImagePath(imageName string) string {
	return path.Join(ImagePath, fmt.Sprintf("%s.tar", imageName))
}

func GetLowerPath(containerId string) string {
	return fmt.Sprintf(lowerPathFormat, containerId)
}

func GetUpperPath(containerId string) string {
	return fmt.Sprintf(upperPathFormat, containerId)
}

func GetWorkPath(containerId string) string {
	return fmt.Sprintf(workPathFormat, containerId)
}

func GetMergedPath(containerId string) string {
	return fmt.Sprintf(mergedPathFormat, containerId)
}

func GetMountOption(containerId string) string {
	lower := GetLowerPath(containerId)
	upper := GetUpperPath(containerId)
	work := GetWorkPath(containerId)
	return fmt.Sprintf(mountOptionFormat, lower, upper, work)
}
