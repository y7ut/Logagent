package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/hpcloud/tail"
	"github.com/spf13/cobra"
	"github.com/y7ut/logagent/conf"
	"gopkg.in/ini.v1"
)

var (
	num  int
	feed bool
)

// VersionCmd represents the version command
var LogCommand = &cobra.Command{
	Use:   "logs",
	Short: "Show logs of Bifrost",
	Run: func(cmd *cobra.Command, args []string) {
		configPath := cmd.Flag("config").Value.String()
		checkconfig(configPath)
		if err := ini.MapTo(conf.APPConfig, configPath); err != nil {
			log.Fatalf("load ini file error: %s ", err)
			return
		}
		logs()
	},
}

func logs() {
	logFile := path.Join(conf.APPConfig.Log.Path, conf.APPConfig.Log.Name)
	loggg, err := os.Open(logFile)
	if err != nil {
		fmt.Println(err)
	}
	stat, _ := loggg.Stat()
	// 获取文件的字节大小
	endOffset := stat.Size()

	if feed && num == 0 {
		num = 5
	}
	PrintlnTaillF(loggg, num)
	// offset := TailNWithScanner(loggg, num)

	if feed {
		config := tail.Config{
			ReOpen:    false, // true则文件被删掉阻塞等待新建该文件，false则文件被删掉时程序结束
			Follow:    true,  // true则一直阻塞并监听指定文件，false则一次读完就结束程序
			Location:  &tail.SeekInfo{Offset: endOffset, Whence: 0},
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

func PrintlnTaillF(logFile *os.File, length int) {
	lines := tailLinesSliceOfFile(logFile, length)
	for k := range lines {
		fmt.Print(lines[len(lines)-k-1])
	}
}

func tailLinesSliceOfFile(logFile *os.File, length int) []string {
	stat, _ := logFile.Stat()
	// 获取文件的字节大小
	endOffset := stat.Size()
	// 记录已经读取的字节数
	var offset int64 = -1
	// 制作一个缓冲区用于从末尾读取文件
	lineBuilder := make([]byte, 0)
	// 记录读取的行
	lines := make([]string, 0, length)
	for endOffset-(-offset) > 0 {
		tmpByte := make([]byte, 1)
		// 从末尾读取文件
		logFile.Seek(offset, io.SeekEnd)
		_, err := logFile.Read(tmpByte)
		if err != nil {
			break
		}
		// offset -1 是文件EOF前的最后一个字节，直接跳过
		if offset == -1 {
			offset--
			continue
		}
		// 如果遇到换行符了，证明到了一行的末尾
		if tmpByte[0] == '\n' {
			if len(lines) >= length {
				break
			}
			// 打包缓冲区，制作一个带\n的字符串，并存入读取的行记录中
			lineBuilder = append(lineBuilder, tmpByte[0])
			lines = append(lines, string(lineBuilder))
			lineBuilder = lineBuilder[:0]
		} else {
			// 将每次读到的字节都写进缓冲区的头部
			lineBuilder = append(tmpByte, lineBuilder...)
		}
		offset--
	}
	return lines
}

func init() {
	LogCommand.Flags().IntVarP(&num, "number", "n", 5, "get last number line of logs")
	LogCommand.Flags().StringP("config", "c", "./bifrost.conf", "config bifrost file")
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
