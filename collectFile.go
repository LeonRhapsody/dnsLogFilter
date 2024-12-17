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

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {

					if strings.HasSuffix(event.Name, ".gz") {

						log.Println("[New] New file found:", event.Name)
						T.FoundFilePath <- event.Name

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

					if strings.HasSuffix(event.Name, ".gz") {

						fileInfo, err := os.Stat(event.Name)
						if err != nil {
							log.Fatal("无法获取文件或目录信息:", err)
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
								T.FoundFilePath <- event.Name
							}
							log.Println("[New] New file found:", event.Name)
							T.FoundFilePath <- event.Name
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

		T.FoundFilePath <- path.Join(root)

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
				T.FoundFilePath <- "done"
			}

		}

	}()
	T.wg.Wait()
}
