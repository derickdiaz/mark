package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type MarkDB interface {
	Get(index int) (string, error)
	Add(path string) error
	List() ([]string, error)
	Clear() error
	Delete(index int) error
}

type LocalMarkDB struct {
	DBFile   string
	filePerm os.FileMode
}

func NewLocalMarkDB() (*LocalMarkDB, error) {
	dbFile, err := GetLocalMarkFile()
	if err != nil {
		return nil, err
	}
	return &LocalMarkDB{DBFile: dbFile, filePerm: 0660}, nil
}

func (l *LocalMarkDB) Get(index int) (string, error) {
	if index < 0 {
		return "", errors.New("invalid index")
	}
	paths, err := l.List()
	if err != nil {
		return "", err
	}
	if index < 0 || index > len(paths)-1 {
		return "", errors.New("invalid index")
	}
	return paths[index], nil
}

func (l *LocalMarkDB) Add(path string) error {
	writtenPaths, err := l.List()
	if err != nil {
		return err
	}
	var paths []string
	paths = append(paths, path)
	paths = append(paths, writtenPaths...)
	l.Clear()
	file, err := os.OpenFile(l.DBFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, l.filePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, item := range paths {
		_, err = file.WriteString(item + "\n")
	}
	return err
}

func (l *LocalMarkDB) List() ([]string, error) {
	file, err := os.OpenFile(l.DBFile, os.O_RDONLY|os.O_CREATE, l.filePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var results []string
	for scanner.Scan() {
		line := scanner.Text()
		results = append(results, line)
	}
	return results, nil
}

func (l *LocalMarkDB) Delete(suppliedIndex int) error {
	paths, err := l.List()
	if err != nil {
		return err
	}
	if suppliedIndex < 0 || suppliedIndex >= len(paths) {
		return errors.New("invalid index")
	}
	l.Clear()
	for index, path := range paths {
		if index == suppliedIndex {
			continue
		}
		l.Add(path)
	}
	return nil
}

func (l *LocalMarkDB) Clear() error {
	return os.Truncate(l.DBFile, 0)
}

func GetLocalMarkFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	markFile := filepath.Join(homeDir, ".mark")
	return markFile, nil
}

type MarkCli struct {
	db MarkDB
}

func NewMarkCli(db MarkDB) (*MarkCli, error) {
	return &MarkCli{db: db}, nil
}

func NewMarkCliWithLocalDB() (*MarkCli, error) {
	db, err := NewLocalMarkDB()
	if err != nil {
		return nil, err
	}
	mark, err := NewMarkCli(db)
	if err != nil {
		return nil, err
	}
	return mark, nil
}

func (m *MarkCli) DisplayHelp(args []string) {
	fmt.Println(`
Marks the current location.
If no command is specified, the current working directory is saved to the mark db.

Usage:
	mark [command]

Available Commands:
	help            Displays help menu
	add             Adds the current working directory to mark db(Default action)
	back   <index>  Prints out the number of directories back based on the index provided
	clear           Clears out the paths in the mark db
	delete <index>  Deletes out a path in mark db based on the index provided
	get    <index>  Get the path in mark db based on the index provided
	list            List out the all the marked paths by index
	install         Prints out directions to create move and back commands in your .bashrc
`)
}

func (m *MarkCli) Back(args []string) {
	cwd, err := os.Getwd()
	m.handleError(err)
	if len(args) != 1 {
		m.handleError(errors.New("invalid number of args"))
	}
	index, err := strconv.Atoi(args[0])
	m.handleError(err)
	arr := strings.Split(cwd, "/")
	if index < 0 {
		m.handleError(errors.New("invalid index"))
	}
	directoriesBack := len(arr) - index
	if directoriesBack == 1 {
		fmt.Println("/")
		return
	} else if directoriesBack <= 0 {
		m.handleError(errors.New("invalid index"))
	}
	fmt.Println(strings.Join(arr[0:directoriesBack], "/"))
}

func (m *MarkCli) List(args []string) {
	if len(args) != 0 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	paths, err := m.db.List()
	m.handleError(err)
	for index, path := range paths {
		fmt.Printf("[%v] %v\n", index, path)
	}
}

func (m *MarkCli) Add(args []string) {
	if len(args) != 0 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	path, err := os.Getwd()
	m.handleError(err)
	paths, err := m.db.List()
	m.handleError(err)
	if !slices.Contains(paths, path) {
		err = m.db.Add(path)
		if err != nil {
			m.handleError(errors.New("invalid number of arguments"))
		}
		return
	}
	fmt.Println("path already exists. Moving to top.")
	m.db.Clear()
	m.db.Add(path)
	for _, item := range paths {
		if item == path {
			continue
		}
		err := m.db.Add(item)
		m.handleError(err)
	}
}

func (m *MarkCli) Get(args []string) {
	if len(args) > 1 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	index := 0
	var err error
	if len(args) == 1 {
		index, err = strconv.Atoi(args[0])
		if err != nil {
			m.handleError(errors.New("index is not a number"))
		}
	}
	path, err := m.db.Get(index)
	m.handleError(err)
	fmt.Println(path)
}

func (m *MarkCli) Install(args []string) {
	fmt.Print(`
Run the following commands to create a move function based on the index provided:

1. Add the following code to ~/.bashrc

move() {
	local readonly DEST=$(mark get $1)
	if [[ ! -z $DEST ]]; then
		cd $DEST
	fi
}

back() {
	local readonly DEST=$(mark back $1)
	if [[ ! -z $DEST ]]; then
		cd $DEST
	fi
}

2. Run the following command
source ~/.bashrc
`)
}

func (m *MarkCli) Clear(args []string) {
	err := m.db.Clear()
	m.handleError(err)
}

func (m *MarkCli) Delete(args []string) {
	if len(args) != 1 {
		m.handleError(errors.New("specify index"))
	}
	index, err := strconv.Atoi(args[0])
	m.handleError(err)
	err = m.db.Delete(index)
	m.handleError(err)
}

func (m *MarkCli) handleError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	mark, err := NewMarkCliWithLocalDB()
	if err != nil {
		panic(err)
	}
	commands := map[string]func(args []string){
		"add":     func(args []string) { mark.Add(args) },
		"back":    func(args []string) { mark.Back(args) },
		"clear":   func(args []string) { mark.Clear(args) },
		"delete":  func(args []string) { mark.Delete(args) },
		"get":     func(args []string) { mark.Get(args) },
		"help":    func(args []string) { mark.DisplayHelp(args) },
		"install": func(args []string) { mark.Install(args) },
		"list":    func(args []string) { mark.List(args) },
	}
	// If no arguments are specified then the default action is to
	// add the current working directory
	args := os.Args
	if len(args) == 1 {
		args = append(args, "add")
	}

	// If the command used is not one that is defined
	// notify the user and display the help menu
	command, ok := commands[args[1]]
	if !ok {
		fmt.Fprintln(os.Stderr, "invalid option. displaying help.")
		command = commands["help"]
	}

	var commandArgs []string
	if len(args) >= 2 {
		commandArgs = args[2:]
	}
	command(commandArgs)
}
