package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lmf112358/moyu-mahjiong-cli/internal/game"
	"github.com/lmf112358/moyu-mahjiong-cli/internal/netplay"
	"github.com/lmf112358/moyu-mahjiong-cli/internal/terminal"
)

const version = "0.8.1"

func main() {
	fmt.Println(`  __  __  ___  __   __ _   _       _ ___   _   _   _  ___ `)
	fmt.Println(` |  \/  |/ _ \ \ \ / /| | | |     | |_ _| /_\ | \ | |/ __|`)
	fmt.Println(` | |\/| | (_) | \ V / | |_| |  _  | || | / _ \|  \| | (_ |`)
	fmt.Println(` |_|  |_|\___/   |_|   \___/   \__/|___/_/ \_\_|\_|_|\___|`)
	fmt.Printf("             摸鱼雀 CLI v%s\n\n", version)
	if len(os.Args) < 2 {
		menu()
		return
	}
	switch os.Args[1] {
	case "play":
		play(os.Args[2:])
	case "host":
		host(os.Args[2:], "")
	case "join":
		join(os.Args[2:], "")
	case "version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "未知命令：", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func menu() {
	r := bufio.NewReader(os.Stdin)
	display := chooseDisplay(r)
	fmt.Println()
	fmt.Println("  1) 四人麻将 · 单机")
	fmt.Println("  2) 三人麻将 · 单机")
	fmt.Println("  3) 创建局域网房间")
	fmt.Println("  4) 加入局域网房间")
	fmt.Print("\n请选择 > ")
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	switch line {
	case "1":
		runLocal(4, "摸鱼人", 1, display)
	case "2":
		runLocal(3, "摸鱼人", 1, display)
	case "3":
		host(nil, display)
	case "4":
		join(nil, display)
	default:
		fmt.Println("已退出。")
	}
}

func play(args []string) {
	fs := flag.NewFlagSet("play", flag.ExitOnError)
	players := fs.Int("players", 4, "玩家数：3 或 4")
	name := fs.String("name", "摸鱼人", "昵称")
	rounds := fs.Int("rounds", 1, "1=东风战，2=半庄")
	display := fs.String("display", "", "牌面：stealth 或 normal")
	fs.Parse(args)
	mustPlayers(*players)
	mode := terminal.ParseDisplayMode(*display)
	if *display == "" {
		mode = chooseDisplay(bufio.NewReader(os.Stdin))
	}
	runLocal(*players, *name, *rounds, mode)
}

func runLocal(players int, name string, rounds int, display terminal.DisplayMode) {
	names := []string{name}
	cs := []game.Controller{terminal.NewControllerWithMode(display)}
	botNames := []string{"摸鱼AI·甲", "摸鱼AI·乙", "摸鱼AI·丙"}
	for i := 1; i < players; i++ {
		names = append(names, botNames[i-1])
		cs = append(cs, game.AI{Level: 1})
	}
	e := game.NewEngine(game.Config{Players: players, Rounds: rounds}, names, cs)
	r := e.Run()
	printRanked(r)
}

func host(args []string, preset terminal.DisplayMode) {
	fs := flag.NewFlagSet("host", flag.ExitOnError)
	players := fs.Int("players", 4, "总玩家数：3 或 4")
	humans := fs.Int("humans", 2, "真人总数（含房主，其余为AI）")
	port := fs.Int("port", 18888, "监听端口")
	name := fs.String("name", "房主", "昵称")
	rounds := fs.Int("rounds", 1, "1=东风战，2=半庄")
	display := fs.String("display", "", "牌面：stealth 或 normal")
	fs.Parse(args)
	mode := preset
	if args == nil {
		r := bufio.NewReader(os.Stdin)
		*players = askInt(r, "玩家数（3/4）", 4)
		*humans = askInt(r, "真人总数（含房主，其余为AI）", 2)
		*port = askInt(r, "端口", 18888)
		*name = ask(r, "昵称", "房主")
		if mode == "" {
			mode = chooseDisplay(r)
		}
	}
	if mode == "" {
		if *display == "" {
			mode = chooseDisplay(bufio.NewReader(os.Stdin))
		} else {
			mode = terminal.ParseDisplayMode(*display)
		}
	}
	mustPlayers(*players)
	if *humans < 2 || *humans > *players {
		fatal("联机至少需要 2 位真人（含房主）；只想和 AI 对战请用 play")
	}
	fmt.Printf("房间已监听 0.0.0.0:%d，等待 %d 位同事加入……\n", *port, *humans-1)
	ln, peers, err := netplay.Listen(fmt.Sprintf(":%d", *port), *humans-1, func(i int, n string) { fmt.Printf("✓ %s 已加入（%d/%d）\n", n, i+1, *humans) })
	if err != nil {
		fatal(err.Error())
	}
	defer ln.Close()
	names := []string{*name}
	cs := []game.Controller{terminal.NewControllerWithMode(mode)}
	for _, p := range peers {
		names = append(names, p.Name)
		cs = append(cs, p)
	}
	for len(names) < *players {
		names = append(names, fmt.Sprintf("摸鱼AI·%d", len(names)))
		cs = append(cs, game.AI{Level: 1})
	}
	e := game.NewEngine(game.Config{Players: *players, Rounds: *rounds}, names, cs)
	result := e.Run()
	for _, p := range peers {
		p.Finish(result)
	}
	printRanked(result)
}

func join(args []string, preset terminal.DisplayMode) {
	fs := flag.NewFlagSet("join", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:18888", "房主地址")
	name := fs.String("name", "同事", "昵称")
	display := fs.String("display", "", "牌面：stealth 或 normal")
	fs.Parse(args)
	mode := preset
	if args == nil {
		r := bufio.NewReader(os.Stdin)
		*addr = ask(r, "房主地址", "127.0.0.1:18888")
		*name = ask(r, "昵称", "同事")
		if mode == "" {
			mode = chooseDisplay(r)
		}
	}
	if mode == "" {
		if *display == "" {
			mode = chooseDisplay(bufio.NewReader(os.Stdin))
		} else {
			mode = terminal.ParseDisplayMode(*display)
		}
	}
	if args != nil {
		host := strings.SplitN(*addr, ":", 2)[0]
		if host == "127.0.0.1" || host == "localhost" || host == "::1" {
			fmt.Println("提示：连接的是本机地址。若房主在别的电脑，请用 --addr 房主IP:端口")
		}
	}
	fmt.Println("正在连接", *addr, "……")
	result, err := netplay.Join(*addr, *name, terminal.NewControllerWithMode(mode))
	if err != nil {
		fatal("连接中断：" + err.Error())
	}
	printRanked(result)
}

func printRanked(r game.MatchResult) {
	sort.SliceStable(r.Players, func(i, j int) bool { return r.Players[i].Score > r.Players[j].Score })
	terminal.PrintResult(os.Stdout, r)
}
func ask(r *bufio.Reader, p, def string) string {
	fmt.Printf("%s [%s] > ", p, def)
	s, _ := r.ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}
func askInt(r *bufio.Reader, p string, def int) int {
	s := ask(r, p, strconv.Itoa(def))
	n, e := strconv.Atoi(s)
	if e != nil {
		return def
	}
	return n
}
func chooseDisplay(r *bufio.Reader) terminal.DisplayMode {
	fmt.Println("请选择麻将显示模式：")
	fmt.Println("  1) 隐匿模式：一/1/① 分别表示万/条/筒")
	fmt.Println("  2) 正常模式：使用放大的数字/花色牌块")
	fmt.Print("显示模式 [1] > ")
	s, _ := r.ReadString('\n')
	return terminal.ParseDisplayMode(s)
}
func mustPlayers(n int) {
	if n != 3 && n != 4 {
		fatal("玩家数只能是 3 或 4")
	}
}
func fatal(s string) { fmt.Fprintln(os.Stderr, "错误：", s); os.Exit(1) }
func usage() {
	fmt.Println(`用法：
	majiang play [--players 3|4] [--rounds 1|2] [--display stealth|normal]
	majiang host [--players 3|4] [--humans N] [--display stealth|normal]
	majiang join [--addr IP:端口] [--display stealth|normal]
  majiang version`)
}
