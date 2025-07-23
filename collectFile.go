package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (T *Tasks) watchSingleDir() {

	//初始化一个监听
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error:", err)
		return
	}
	defer watcher.Close()

	fileInfo, err := os.Stat(T.InputDir)
	if err == nil && fileInfo.IsDir() {

		// add单个
		log.Println("add a New WorkDir:", T.InputDir)
		watcher.Add(T.InputDir)

	} else {
		panic(fmt.Errorf(T.InputDir, "is not exist or ", err))
	}

	//离线文件分析
	err = filepath.WalkDir(T.InputDir, func(root string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err) // 可能会有访问权限等错误
			return nil
		}

		if d.IsDir() {
			return nil // 跳过目录
		}

		if strings.HasSuffix(d.Name(), T.LogType) {
			Info, _ := d.Info()
			if time.Now().Sub(Info.ModTime()).Hours() <= 2 {
				log.Println("[New] New file (old) found:", path.Join(root))
				T.NewFilePath <- path.Join(root)
			}
		}

		return nil
	})
	if err != nil {
		log.Println(err)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {

					if strings.HasSuffix(event.Name, T.LogType) {

						log.Println("[New] New file found:", event.Name)
						T.NewFilePath <- event.Name

					}

				}
			case err := <-watcher.Errors:
				log.Fatal("Error:", err)
			}
		}
	}()

	<-done
}

func (T *Tasks) watchMultipleDir() {

	//确认启动当前的日期目录
	today := time.Now().Format("20060102")
	todayWorkDir := path.Join(T.InputDir, today)

	//初始化一个监听
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error:", err)
		return
	}
	defer watcher.Close()

	fileInfo, err := os.Stat(T.InputDir)
	if err == nil && fileInfo.IsDir() {

		// add单个
		log.Println("add a New WorkDir:", T.InputDir)
		watcher.Add(T.InputDir)

		//add 父目录，同时监听可能创建的新的日期目录
		err = filepath.Walk(T.InputDir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				log.Println("add a New WorkDir:", path)
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			log.Fatal("Error:", err)
			return
		}
	} else {
		panic(fmt.Errorf(T.InputDir, "is not exist or ", err))
	}

	fileInfo, err = os.Stat(todayWorkDir)
	if err == nil && fileInfo.IsDir() {
		//add 现在的日期目录，开启今日日志文件的监听
		err = filepath.Walk(todayWorkDir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				log.Println("add a New WorkDir:", path)
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			log.Fatal("Error:", err)
			return
		}
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {

					if strings.HasSuffix(event.Name, T.LogType) {

						fileInfo, err := os.Stat(event.Name)
						if err != nil {
							log.Println("无法获取文件或目录信息:", err)
						}

						//只监听今日目录，同时remove3天前的目录清单
						if fileInfo.IsDir() {
							log.Println("find a New WorkDir: ", event.Name)
							today := time.Now().Format("20060102")
							last3DaysWorkDir := path.Join(T.InputDir, time.Now().AddDate(0, 0, -3).Format("20060102"))

							if strings.HasSuffix(event.Name, today) {
								watcher.Add(event.Name)
								log.Println("add a New WorkDir: ", event.Name)

								for _, watchName := range watcher.WatchList() {
									if watchName == last3DaysWorkDir {
										watcher.Remove(last3DaysWorkDir)
										log.Println("remove 3 days before WorkDir: ", last3DaysWorkDir)
									}
								}

								log.Println("当前监听列表为", watcher.WatchList())
							}

						} else {
							if !strings.HasSuffix(event.Name, ".CHK") && !strings.HasSuffix(event.Name, ".AUDIT") {
								log.Println("New file found:", event.Name)
								T.NewFilePath <- event.Name
							}
							log.Println("[New] New file found:", event.Name)
							T.NewFilePath <- event.Name
							//}

						}

					}
				}
			case err := <-watcher.Errors:
				log.Fatal("Error:", err)
			}
		}
	}()

	<-done
}

func (T *Tasks) offlineWatch() {

	err := filepath.WalkDir(T.InputDir, func(root string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err) // 可能会有访问权限等错误
			return nil
		}

		if d.IsDir() {
			return nil // 跳过目录
		}

		if strings.HasSuffix(d.Name(), T.LogType) {
			T.NewFilePath <- path.Join(root)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		timer := time.NewTicker(1000 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				T.NewFilePath <- "done"
			}

		}

	}()
	T.wg.Wait()
}

func (T *Tasks) recoverLatestTempFile() {

	// 收集所有匹配的 .gz.tmp 文件
	for _, task := range T.TaskInfos {
		var tempFiles []string

		log.Printf("Scanning for latest .gz.tmp file in %s ", task.OutputDir)
		err := filepath.WalkDir(task.OutputDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				log.Printf("Error accessing path %s: %v", path, err)
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".gz.tmp") {

				tempFiles = append(tempFiles, path)
			}
			return nil
		})
		if err != nil {
			log.Printf("Error scanning directory %s: %v", task.OutputDir, err)
			return
		}
		// 重命名并上传最新文件
		for _, file := range tempFiles {
			gzFile := strings.TrimSuffix(file, ".tmp")
			log.Printf("[Recover] Renaming latest temp file %s to %s", file, gzFile)
			if err := os.Rename(file, gzFile); err != nil {
				log.Fatalf("Error renaming file %s: %v", file, err)
			}
			if task.Upload.IsUpload {
				err = task.uploadFile(gzFile)
				if err != nil {
					log.Printf("[Error Upload] failed: %v", err)
				} else {
					log.Printf("[Upload] %s to sftp %s successfully", gzFile, task.Upload.SFTPHost)
					T.deleteFile(gzFile)
				}
			}
		}

	}

}
