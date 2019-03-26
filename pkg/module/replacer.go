package module

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/gomods/athens/pkg/config"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getDirList(dirpath string) ([]string, error) {
	var dir_list []string
	dir_err := filepath.Walk(dirpath,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				dir_list = append(dir_list, path)
				return nil
			}

			return nil
		})
	return dir_list, dir_err
}

func contextReplace(fileName, src, dst string) error {
	in, err := os.Open(fileName)
	if err != nil {
		fmt.Println("open file fail:", err)
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(fileName+".bak", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Open write file fail:", err)
		return err
	}
	defer out.Close()

	reader := bufio.NewReader(in)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("ReadLine err:", err)
			return err
		}
		newLine := strings.Replace(string(line), src, dst, -1)
		_, err = out.WriteString(newLine + "\n")
		if err != nil {
			fmt.Println("write to file fail:", err)
			return err
		}
	}
	//os.Remove(fileName)
	err = os.Rename(fileName+".bak", fileName)
	if err != nil {
		fmt.Println("rename err:", err)
		return err
	}
	return nil
}

func convert(m goModule) goModule {
	if config.GetModConfig() == nil {
		fmt.Println("config.GetModConfig():", config.GetModConfig())
		return m
	}

	modMap := config.GetModConfig()

	for key, value := range modMap {
		if strings.HasPrefix(m.Path, value) {

			cmd := exec.Command("unzip", m.Zip)
			n := strings.LastIndex(m.Zip, "/")
			if n <= 0 {
				fmt.Println("path is invalid:")
				return m
			}
			prefix := m.Zip[:n+1] //save dir
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.Dir = prefix
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err := cmd.Run()
			if err != nil {
				fmt.Println("run err:", err)
				return m
			}
			os.Remove(m.Zip)

			srcDir := prefix + value + "@" + m.Version
			desDir := prefix + key + "@" + m.Version

			//list,err:=getDirList(m.Zip[:n+1])
			//if err != nil {
			//	return m
			//}

			err = CopyDir(srcDir, desDir)
			if err != nil {
				fmt.Println("copy dir err:", err)
				return m
			}

			cmd = exec.Command("zip", "-rD", m.Zip[n+1:], key+"@"+m.Version)

			stdout = &bytes.Buffer{}
			stderr = &bytes.Buffer{}
			cmd.Dir = prefix
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err = cmd.Run()
			if err != nil {
				stdout.String()
				fmt.Println("cmd run err:", err, " out:", stdout.String(), "stderr:", stderr.String())
				return m
			}
			err = contextReplace(m.GoMod, value, key)
			if err != nil {
				fmt.Println("contextReplace err:", err)
				return m
			}
			m.Path = strings.Replace(m.Path, value, key, -1) //todo

			break
		}
	}
	return m
}

func convertReplace(m goModule) goModule {
	if config.GetModConfig() == nil {
		return m
	}

	modMap := config.GetModConfig()
	for key, value := range modMap {
		if strings.HasPrefix(m.Path, value) {
			err := contextReplace(m.GoMod, key, value)
			if err != nil {
				fmt.Println("contextReplace err:", err)
				return m
			}
			break
		}
	}
	return m
}

func CopyDir(srcPath string, destPath string) error {
	//检测目录正确性
	if srcInfo, err := os.Stat(srcPath); err != nil {
		fmt.Println(err.Error())
		return err
	} else {
		if !srcInfo.IsDir() {
			return fmt.Errorf("srcPath is not dir")
		}
	}
	if destInfo, err := os.Stat(destPath); err != nil {
		os.MkdirAll(destPath, 0777)
	} else {
		if !destInfo.IsDir() {
			return fmt.Errorf("destInfo is not dir")
		}
	}

	err := filepath.Walk(srcPath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if !f.IsDir() {
			destNewPath := strings.Replace(path, srcPath, destPath, -1)
			moveFile(path, destNewPath)
		}
		return nil
	})
	if err != nil {
		fmt.Printf(err.Error())
	}
	return err
}

//生成目录并移动文件
func moveFile(src, dest string) (w int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer srcFile.Close()
	//分割path目录
	destSplitPathDirs := strings.Split(dest, "/")

	//检测时候存在目录
	destSplitPath := ""
	for index, dir := range destSplitPathDirs {
		if index < len(destSplitPathDirs)-1 {
			destSplitPath = destSplitPath + dir + "/"
			b, _ := pathExists(destSplitPath)
			if b == false {
				//创建目录
				err := os.Mkdir(destSplitPath, os.ModePerm)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
	dstFile, err := os.Create(dest)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}

//检测文件夹路径时候存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func replace(mod string) (string, bool) {
	if config.GetModConfig() == nil {
		return mod, false
	}

	modMap := config.GetModConfig()
	res, ok := modMap[mod]
	if !ok {
		for key, value := range modMap {
			if strings.HasPrefix(mod, key) && strings.HasPrefix(mod, key+"/") {
				mod = strings.Replace(mod, key, value, -1)
				return mod, true
			}
		}
		return mod, false
	}

	return res, true
}
