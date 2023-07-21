package cmd

import (
	"bufio"
	"container/list"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hpcloud/tail"
	"github.com/spf13/cobra"
)

var (
	logFile = "./runtime/log/bifrost.log"
	num     int
	feed    bool
)

// VersionCmd represents the version command
var LogCommand = &cobra.Command{
	Use:   "logs",
	Short: "Show logs of Bifrost",
	Run: func(cmd *cobra.Command, args []string) {
		logs()
	},
}

func logs() {
	loggg, err := os.Open(logFile)
	if err != nil {
		fmt.Println(err)
	}
	scanner := bufio.NewScanner(loggg)
	stack := list.New()
	offset := int64(0)

	if feed && num == 0 {
		num = 5
	}

	box := make(chan string, num)
	go func() {
		for {
			t := <-box
			stack.PushBack(t)
			if num != 0 && stack.Len() > num {
				stack.Remove(stack.Front())
			}
		}
	}()

	for scanner.Scan() {
		box <- scanner.Text()
		offset += int64(len(scanner.Bytes()) + 1)
		// 因为有换行符,所以需要+1
	}

	for stack.Front() != nil {
		fmt.Println(stack.Front().Value)
		stack.Remove(stack.Front())
	}

	if feed {
		config := tail.Config{
			ReOpen:    false, // true则文件被删掉阻塞等待新建该文件，false则文件被删掉时程序结束
			Follow:    true,  // true则一直阻塞并监听指定文件，false则一次读完就结束程序
			Location:  &tail.SeekInfo{Offset: offset, Whence: 0},
			MustExist: true, // true则没有找到文件就报错并结束，false则没有找到文件就阻塞保持住
			Poll:      true, // 使用Linux的Poll函数，poll的作用是把当前的文件指针挂到等待队列
			Logger:    tail.DiscardingLogger,
		}
		tailer, err := tail.TailFile(logFile, config)

		if err != nil {
			fmt.Println(err)
			return
		}

		go func() {
			for {
				line := <-tailer.Lines
				fmt.Println(line.Text)
			}
		}()

		for s := range sign() {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
				// log.Println("Safe Exit:", s)
				return
			}
		}
	}
}

func init() {
	LogCommand.Flags().IntVarP(&num, "number", "n", 0, "get last number line of logs")
	LogCommand.Flags().BoolVarP(&feed, "feed", "f", false, "feed logs")
	RootCmd.AddCommand(LogCommand)
}

func sign() <-chan os.Signal {
	c := make(chan os.Signal, 2)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}

	// 监听信号, 判断是否忽略 sighup 信号量
	if !signal.Ignored(syscall.SIGHUP) {
		signals = append(signals, syscall.SIGHUP)
	}

	signal.Notify(c, signals...)

	return c
}
