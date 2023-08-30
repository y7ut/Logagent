package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	pidFile string
)

var startCmd = &cobra.Command{
	Use:   "start  [--config config]",
	Short: "Start bifrost in the background process",
	Long:  "Start bifrost in the background process",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startProcess(cmd, args)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "leave bifrost process",
	Long:  "Stop bifrost process in the background process",
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopProcess()
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [--config config]",
	Short: "Restart the bifrost process",
	Long:  "Restart the bifrost process that is running as a daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := stopProcess()
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
		return startProcess(cmd, args)
	},
}

func startProcess(cmd *cobra.Command, args []string) error {
	pid, err := daemonStart()
	if err != nil {
		return err
	}
	if pid != -1 {
		return fmt.Errorf("failed to start Bifrost: Bifrost is already exists in pid[%d]", pid)
	}

	configPath := cmd.Flag("config").Value.String()
	checkconfig(configPath)

	osArg := os.Args
	osArg[1] = "run"
	runnerCmd := &exec.Cmd{
		Path: osArg[0],
		Args: osArg,
		Env:  os.Environ(),
	}

	logStd, err := os.OpenFile(filepath.Join(filepath.Dir(pidFile), "start.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(os.Getpid(), ": failed to open start log file:", err)
	}
	runnerCmd.Stderr = logStd
	runnerCmd.Stdout = logStd

	runnerCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = runnerCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start Bifrost: %s", err)
	}

	// <-time.After(5 * time.Second)
	err = os.WriteFile(pidFile, []byte(strconv.Itoa(runnerCmd.Process.Pid)), 0666)
	if err != nil {
		return fmt.Errorf("failed to record pid, you may not be able to stop the program with `./bifrost stop`")
	}
	fmt.Println("🎏 bifrost started...")

	return nil
}

func stopProcess() error {
	pid, err := daemonStart()
	if err != nil {
		return err
	}
	if pid == -1 {
		return fmt.Errorf("bifrost is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {

		return fmt.Errorf("failed to find process by pid: %d, reason: %v", pid, process)
	}
	err = process.Signal(os.Interrupt)
	if err != nil {
		err = os.Remove(pidFile)
		if err != nil {
			return fmt.Errorf("failed to remove pid file")
		}
		return fmt.Errorf("failed to kill process %d: %v", pid, err)
	}

	err = os.Remove(pidFile)
	if err != nil {
		return fmt.Errorf("failed to remove pid file")
	}

	fmt.Println("🤚 bifrost paused...")
	return nil
}

func daemonStart() (int, error) {
	var pid = -1
	// 获取当前目录
	ex, err := os.Executable()
	if err != nil {
		return pid, err
	}
	exPath := filepath.Dir(ex)
	_ = os.MkdirAll(filepath.Join(exPath, "runtime"), 0700)
	pidFile = filepath.Join(exPath, "runtime/pid")

	if _, err := os.Stat(pidFile); err != nil {
		if os.IsNotExist(err) {
			return pid, nil
		}
	}

	content, err := os.ReadFile(pidFile)
	if err != nil {
		log.Fatalf("Failed to read pid file : %s", err)
		return -1, err
	}
	pid, err = strconv.Atoi(string(content))
	if err != nil {
		log.Fatalf("Failed to parse pid file : %s", err)
		return -1, err
	}
	return pid, nil
}

func init() {
	startCmd.Flags().StringP("config", "c", "./bifrost.conf", "config bifrost file")
	restartCmd.Flags().StringP("config", "c", "./bifrost.conf", "config bifrost file")
	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(stopCmd)
	RootCmd.AddCommand(restartCmd)
}
