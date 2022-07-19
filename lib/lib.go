package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

type Composition struct {
	Services []service
	Files    []file
	Packages []deb_package
}
type service struct {
	ServiceName    string
	Action         string
	RestartCommand string
}

type file struct {
	FileLocation string
	FileName     string
	Group        string
	Permissions  int
	User         string
	Service      string
}

type deb_package struct {
	PackageName string
	Action      string
	Service     string
}

func GetComposition(fileLocation string) Composition {
	yfile, err := ioutil.ReadFile(fileLocation)

	if err != nil {

		log.Fatal(err)
	}

	data := make(map[interface{}]interface{})

	err2 := yaml.Unmarshal(yfile, &data)

	if err2 != nil {

		log.Fatal(err2)
	}

	var comp Composition
	for k, v := range data {

		if k == "files" {
			var f file
			output := make(map[interface{}]interface{})
			mapstructure.Decode(v, &output)
			for _, y := range output {
				mapstructure.Decode(y, &f)
				comp.Files = append(comp.Files, f)
			}
		} else if k == "services" {
			var f service
			output := make(map[interface{}]interface{})
			mapstructure.Decode(v, &output)
			for _, y := range output {
				mapstructure.Decode(y, &f)
				comp.Services = append(comp.Services, f)
			}
		} else if k == "packages" {
			var f deb_package
			output := make(map[interface{}]interface{})
			mapstructure.Decode(v, &output)
			for _, y := range output {
				mapstructure.Decode(y, &f)
				comp.Packages = append(comp.Packages, f)
			}
		}
	}

	return comp
}

func SetFileOwnership(pathToFile, userId, groupId string) {
	u, err := nameGroupToUid(userId, "")
	g, err2 := nameGroupToUid("", groupId)

	if err != nil {
		fmt.Println(err)
	} else if err2 != nil {
		fmt.Println(err2)
	} else {
		err := os.Chown(pathToFile, *u, *g)
		if err != nil {
			fmt.Printf("%s", err)
		} else {
			fmt.Printf("File: %s Permisions updated to:", pathToFile)
		}
	}

}

func PlaceFile(fileName string, fileDestination string) {
	srcFile, err := os.Open(fmt.Sprintf("/opt/maestro/files/%s", fileName))
	checkErr(err)
	defer srcFile.Close()

	destFile, err := os.Create(fileDestination)
	checkErr(err)
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	checkErr(err)

	err = destFile.Sync()
	checkErr(err)

}

func CompareFile(fileName string, fileDestination string) bool {
	compFile, err1 := ioutil.ReadFile(fmt.Sprintf("/opt/maestro/files/%s", fileName))

	if err1 != nil {
		log.Fatal(err1)
	}

	placedFile, err2 := ioutil.ReadFile(fileDestination)

	if err2 != nil {
		fmt.Println(fmt.Sprintf("File not found at %s", fileDestination))
		return false
	}

	return bytes.Equal(compFile, placedFile)
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("Error : %s", err.Error())
	}
}

func SetFilePermissions(pathToFile string, mode os.FileMode) {
	err := os.Chmod(pathToFile, mode)
	if err != nil {
		fmt.Printf("%s", err)
	} else {
		fmt.Printf("File: %s Permisions updated to:%d", pathToFile, mode)
	}
}

func InstallPackage(packageName string) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	if !CheckPackageInstalled(packageName) {
		cmd := exec.Command("apt-get", "install", "-y", "-q", packageName)
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			return
		}
	} else {
		fmt.Println(out)
	}

}

func RemovePackage(packageName string) {
	if CheckPackageInstalled(packageName) {
		out, err := exec.Command("apt-get", "remove", packageName, "-f").Output()
		if err != nil {
			fmt.Printf("%s", err)
		}
		fmt.Println(out)
	} else {
		fmt.Println(fmt.Sprintf("Package %s is not installed", packageName))
	}
}

func GetServiceStatus(servicename string) string {
	cmd := exec.Command("service", servicename, "status")
	out, err := cmd.CombinedOutput()

	fmt.Println(out)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("systemctl finished with non-zero: %v\n", exitErr)
		} else {
			fmt.Printf("failed to run systemct: %v", err)
		}
		return "Service not found"
	}
	return "running"
}

func RestartService(servicename string, rsc string) {
	var restartString string
	fmt.Println(rsc)
	if rsc != "none" {
		restartString = rsc

	} else {
		restartString = fmt.Sprintf("service %s restart", servicename)
	}
	if CheckPackageInstalled(servicename) {
		splitString := strings.Split(restartString, " ")
		out, err := exec.Command(splitString[0], splitString[1:]...).Output()
		if err != nil {
			fmt.Printf("%s", err)
		}
		println(out)
	} else {
		println(fmt.Sprintf("Service %s not found", servicename))
	}
}

func CheckPackageInstalled(packagename string) bool {

	pkgPath, err := exec.LookPath(packagename)
	if err != nil {
		fmt.Printf("%s", err)
		return false
	} else {
		println(fmt.Sprintf("Package %s found at %s", packagename, pkgPath))
		return true
	}
}

func nameGroupToUid(name, group string) (*int, error) {
	if name != "" {
		tmp, e := user.Lookup(name)
		if e != nil {
			fmt.Printf("%s", e)
			return nil, e
		} else {
			x, e := strconv.Atoi(tmp.Uid)
			if e != nil {
				fmt.Printf("%s", e)
				return nil, e
			} else {
				return &x, nil
			}
		}
	}
	if group != "" {
		tmp, e := user.LookupGroup(group)
		if e != nil {
			fmt.Printf("%s", e)
			return nil, e
		} else {
			x, e := strconv.Atoi(tmp.Gid)
			if e != nil {
				fmt.Printf("%s", e)
				return nil, e
			} else {
				return &x, nil
			}
		}
	}

	return nil, nil
}

func AddServiceToRestartList(servicename string, restartList []string) []string {
	var tmp []string

	if len(restartList) < 1 {
		tmp = append(tmp, servicename)
	} else {
		for _, x := range restartList {
			var add2List bool = true
			for _, y := range tmp {
				if x == y {
					add2List = false
				}
			}
			if add2List {
				tmp = append(tmp, x)
			}
		}

	}
	return tmp
}

func CheckFilePermissions(filePath string, m os.FileMode) bool {
	fi, err := os.Lstat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	if m == fi.Mode().Perm() {
		return true
	}

	return false
}

func CheckOwner(filePath string, uid string) bool {
	file_info, err := os.Stat(filePath)
	file_sys := file_info.Sys()
	file_uid := fmt.Sprint(file_sys.(*syscall.Stat_t).Uid)
	if err != nil {
		log.Fatal(err)
	}
	rid, err := nameGroupToUid(uid, " ")
	if err != nil {
		log.Fatal(err)
	}

	frid, err := strconv.Atoi(file_uid)
	if err != nil {
		log.Fatal(err)
	}

	if frid == *rid {
		return true
	}

	return false
}

func CheckGroup(filePath string, gid string) bool {
	file_info, err := os.Stat(filePath)
	file_sys := file_info.Sys()
	file_uid := fmt.Sprint(file_sys.(*syscall.Stat_t).Gid)
	if err != nil {
		log.Fatal(err)
	}
	rid, err := nameGroupToUid(gid, " ")
	if err != nil {
		log.Fatal(err)
	}

	frid, err := strconv.Atoi(file_uid)
	if err != nil {
		log.Fatal(err)
	}

	if frid == *rid {
		return true
	}

	return false
}
