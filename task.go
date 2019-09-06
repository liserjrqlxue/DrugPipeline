package main

import (
	"github.com/liserjrqlxue/simple-util"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type Task struct {
	TaskName       string
	TaskType       string
	TaskScript     string
	TaskArgs       []string
	TaskInfo       map[string]string
	TaskToChan     map[string]map[string]*chan string
	TaskFrom       []*Task
	First, End     bool
	Scripts        map[string]string
	BatchScript    string
	BarcodeScripts map[string]string
	mem            string
	thread         string
	submitArgs     []string
}

func createStartTask() *Task {
	return &Task{
		TaskName:   "Start",
		TaskToChan: make(map[string]map[string]*chan string),
		First:      true,
	}
}

func createEndTask() *Task {
	return &Task{
		TaskName:   "End",
		TaskToChan: make(map[string]map[string]*chan string),
		End:        true,
	}
}

func createTask(cfg map[string]string, local string, submitArgs []string) *Task {
	task := Task{
		TaskName:       cfg["name"],
		TaskInfo:       cfg,
		TaskType:       cfg["type"],
		TaskScript:     filepath.Join(local, "script", cfg["name"]+".sh"),
		TaskArgs:       strings.Split(cfg["args"], ","),
		TaskToChan:     make(map[string]map[string]*chan string),
		Scripts:        make(map[string]string),
		BarcodeScripts: make(map[string]string),
		mem:            cfg["mem"],
		thread:         cfg["thread"],
		submitArgs:     append(submitArgs, "-l", "vf="+cfg["mem"]+"G,p="+cfg["thread"]),
		End:            true,
	}
	if cfg["submitArgs"] != "" {
		task.submitArgs = append(task.submitArgs, sep.Split(cfg["submitArgs"], -1)...)
	}
	return &task
}

func (task *Task) Start() {
	for taskName, chanMap := range task.TaskToChan {
		log.Printf("%-7s -> Task[%-7s]", task.TaskName, taskName)
		for sampleID := range chanMap {
			ch := chanMap[sampleID]
			log.Printf("Task[%-7s:%s] -> Task[%-7s:%s]", task.TaskName, sampleID, taskName, sampleID)
			go func(ch *chan string) { *ch <- "" }(ch)
		}
	}
}

func (task *Task) WaitEnd() {
	for _, fromTask := range task.TaskFrom {
		for taskName, chanMap := range fromTask.TaskToChan {
			for sampleID := range chanMap {
				log.Printf("Task[%-7s:%s] <- %s", taskName, sampleID, <-*chanMap[sampleID])
			}
		}
		log.Printf("%-7s <- Task[%-7s]", task.TaskName, fromTask.TaskName)
	}
}

func (task *Task) WaitFrom(sampleIDs []string) string {
	var hjid = make(map[string]bool)
	for _, fromTask := range task.TaskFrom {
		for _, sampleID := range sampleIDs {
			ch := fromTask.TaskToChan[task.TaskName][sampleID]
			jid := <-*ch
			if jid != "" {
				hjid[jid] = true
			}
		}
	}
	var jid []string
	for id := range hjid {
		jid = append(jid, id)
	}
	return strings.Join(jid, ",")
}

func (task *Task) Run(script, hjid, jobName, jid string) string {
	switch *mode {
	case "sge":
		jid = simple_util.SGEsubmit([]string{script}, hjid, task.submitArgs)
	default:
		throttle <- true
		log.Printf("Run Task[%-7s:%s]:%s", task.TaskName, jobName, script)
		if *dryRun {
			time.Sleep(10 * time.Second)
		} else {
			simple_util.CheckErr(simple_util.RunCmd("bash", script))
		}
		<-throttle
	}
	return jid
}

func (task *Task) SetEnd(info Info, jobName, jid string, sampleList []string) {
	for _, chanMap := range task.TaskToChan {
		for _, sampleID := range sampleList {
			log.Printf("Task[%-7s:%s] -> {%s}", task.TaskName, sampleID, jid)
			*chanMap[sampleID] <- jid
		}
	}
}

func (task *Task) RunTask(info Info, jobName, script string, sampleList []string) {
	var hjid = task.WaitFrom(sampleList)
	log.Printf("Task[%-7s:%s] <- {%s}", task.TaskName, jobName, hjid)
	var jid = task.TaskName + "[" + jobName + "]"
	jid = task.Run(script, hjid, jobName, jid)
	task.SetEnd(info, jobName, jid, sampleList)
}

func (task *Task) CreateScripts(info Info) {
	switch task.TaskType {
	case "sample":
		task.createSampleScripts(info)
	case "batch":
		task.createBatchScripts(info)
	case "barcode":
		task.createBarcodeScripts(info)
	}
}

func (task *Task) createSampleScripts(info Info) {
	for sampleID, sampleInfo := range info.SampleMap {
		script := filepath.Join(*outDir, sampleID, "shell", task.TaskName+".sh")
		task.Scripts[sampleID] = script
		var appendArgs []string
		appendArgs = append(appendArgs, *outDir, *localpath, sampleID)
		for _, arg := range task.TaskArgs {
			switch arg {
			default:
				appendArgs = append(appendArgs, sampleInfo.info[arg])
			}
		}
		createShell(script, task.TaskScript, appendArgs...)
	}
}

func (task *Task) createBatchScripts(info Info) {
	script := filepath.Join(*outDir, "shell", task.TaskName+".sh")
	task.BatchScript = script
	var appendArgs []string
	appendArgs = append(appendArgs, *outDir, *localpath)
	for _, arg := range task.TaskArgs {
		switch arg {
		case "list":
			appendArgs = append(appendArgs, filepath.Join(*outDir, "input.list"))
		default:
		}
	}
	createShell(script, task.TaskScript, appendArgs...)
}

func (task *Task) createBarcodeScripts(info Info) {
	for barcode, barcodeInfo := range info.BarcodeMap {
		script := filepath.Join(*outDir, "shell", strings.Join([]string{barcode, task.TaskName, "sh"}, "."))
		task.BarcodeScripts[barcode] = script
		var appendArgs []string
		appendArgs = append(appendArgs, *outDir, *localpath)
		for _, arg := range task.TaskArgs {
			switch arg {
			case "barcode":
				appendArgs = append(appendArgs, barcode)
			case "fq1":
				appendArgs = append(appendArgs, barcodeInfo.fq1)
			case "fq2":
				appendArgs = append(appendArgs, barcodeInfo.fq2)
			case "list":
				appendArgs = append(appendArgs, barcodeInfo.list)
			}
		}
		createShell(script, task.TaskScript, appendArgs...)
	}
}
