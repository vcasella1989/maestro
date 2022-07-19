package main

import (
	"fmt"
	"io/fs"
	"maestro/lib"
)

func main() {
	//in a non POC composition file location should either be set to default location or changable via a config file
	composition := lib.GetComposition("/opt/maestro/config/composition.yaml")
	//os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	//Install/Uninstall Packages
	for _, x := range composition.Packages {

		isInstalled := lib.CheckPackageInstalled(x.Service)
		if x.Action == "install" {
			if !isInstalled {
				lib.InstallPackage(x.PackageName)
			}
		} else if isInstalled && x.Action == "uninstall" {
			lib.RemovePackage(x.PackageName)
		}
	}

	var servicesRestart []string
	//Place and modify files
	for _, x := range composition.Files {
		rss := false
		//Check if composition file and the file in place are the same, if they are, ignore, if not, place
		if !lib.CompareFile(x.FileName, x.FileLocation) {
			lib.PlaceFile(x.FileName, x.FileLocation)
			rss = true
		}

		if !lib.CheckFilePermissions(x.FileLocation, fs.FileMode(x.Permissions)) {
			lib.SetFilePermissions(x.FileLocation, fs.FileMode(x.Permissions))
			servicesRestart = lib.AddServiceToRestartList(x.Service, servicesRestart)
			rss = true
		}

		if !lib.CheckOwner(x.FileLocation, x.User) || !lib.CheckGroup(x.FileLocation, x.Group) {
			lib.SetFileOwnership(x.FileLocation, x.User, x.Group)
			servicesRestart = lib.AddServiceToRestartList(x.Service, servicesRestart)
			rss = true
		}

		if rss {
			servicesRestart = lib.AddServiceToRestartList(x.Service, servicesRestart)
		}
	}
	for _, x := range servicesRestart {
		rsc := "none"
		fmt.Println(x)
		//Check if the service has a special restart condition
		for _, y := range composition.Services {
			if x == y.ServiceName {
				rsc = y.RestartCommand
			}
		}
		lib.RestartService(x, rsc)
	}
}
