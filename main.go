package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Mark struct {
	Path  string `json:"path"`
	Alias string `json:"alias"`
}

type MarkDB interface {
	GetByAlias(alias string) (*Mark, error)
	GetByIndex(index int) (*Mark, error)
	Add(mark *Mark) error
	List() ([]*Mark, error)
	Clear() error
	Switch(source, dest int) error
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

func (l *LocalMarkDB) GetByAlias(alias string) (mark *Mark, err error) {
	marks, err := l.List()
	if err != nil {
		return
	}
	for _, mark = range marks {
		if mark.Alias == alias {
			return
		}
	}
	err = fmt.Errorf("unable to find mark with alias: %v", alias)
	return
}

func (l *LocalMarkDB) GetByIndex(index int) (mark *Mark, err error) {
	if index < 0 {
		err = errors.New("invalid index")
		return
	}
	marks, err := l.List()
	if err != nil {
		return
	}
	if index < 0 || index > len(marks)-1 {
		err = errors.New("invalid index")
		return
	}
	return marks[index], nil
}

func (l *LocalMarkDB) Add(mark *Mark) (err error) {
	marks, err := l.List()
	if err != nil {
		return
	}
	for _, tmark := range marks {
		if mark.Path == tmark.Path {
			err = errors.New("mark already exists")
			return
		}
		if mark.Alias != "" && mark.Alias == tmark.Alias {
			err = errors.New("alias already exists")
			return
		}
	}
	marks = append(marks, mark)
	jsonData, err := json.MarshalIndent(marks, "", " ")
	if err != nil {
		return
	}
	err = os.WriteFile(l.DBFile, jsonData, 0660)
	return
}

func (l *LocalMarkDB) List() (marks []*Mark, err error) {
	file, err := os.OpenFile(l.DBFile, os.O_RDONLY|os.O_CREATE, l.filePerm)
	if err != nil {
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&marks)
	return
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
	return os.WriteFile(l.DBFile, []byte("[]"), 0660)
}

func (l *LocalMarkDB) Switch(source, dest int) error {
	items, err := l.List()
	if err != nil {
		return err
	}
	if source < 0 || source > len(items)-1 {
		return errors.New("invalid source index")
	}
	if dest < 0 || dest > len(items)-1 {
		return errors.New("invalid dest index")
	}
	sourceItem := items[source]
	destItem := items[dest]
	items[dest] = sourceItem
	items[source] = destItem
	l.Clear()
	for _, mark := range items {
		l.Add(mark)
	}
	return nil
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
    add                         Adds the current working directory to mark db(Default action)
    back    <index>             Prints out the number of directories back based on the index provided
    clear                       Clears out the paths in the mark db
    delete  <index>             Deletes out a path in mark db based on the index provided
    forward <regex>             Looks foward for directories that match a regex
    get     <index>             Get the path in mark db based on the index provided
    help                        Displays help menu
    install                     Prints out directions to create move and back commands in your .bashrc
    jump    <index>             Prints out the number of directories jumping forward from the beginning
    list                        List out the all the marked paths by index
    switch  <source> <dest>     Switch stored paths by their index`)
}

func (m *MarkCli) Switch(args []string) {
	if len(args) != 2 {
		m.handleError(errors.New("invalid number of args"))
	}
	source, err := strconv.Atoi(args[0])
	if err != nil {
		m.handleError(errors.New("source and dest must be an integer"))
	}
	dest, err := strconv.Atoi(args[1])
	if err != nil {
		m.handleError(errors.New("source and dest must be an integer"))
	}
	err = m.db.Switch(source, dest)
	if err != nil {
		m.handleError(err)
	}
	items, err := m.db.List()
	if err != nil {
		m.handleError(err)
	}
	for index, item := range items {
		fmt.Printf("[%v] %v\n", index, item)
	}
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

func (m *MarkCli) Jump(args []string) {
	cwd, err := os.Getwd()
	m.handleError(err)
	if len(args) != 1 {
		m.handleError(errors.New("invalid number of args"))
	}
	index, err := strconv.Atoi(args[0])
	m.handleError(err)
	arr := strings.Split(cwd, "/")
	if index < 0 || index > len(arr)-1 {
		m.handleError(errors.New("invalid index"))
	}
	fmt.Println(strings.Join(arr[0:index+1], "/"))
}

func (m *MarkCli) List(args []string) {
	if len(args) != 0 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	marks, err := m.db.List()
	m.handleError(err)
	for index, mark := range marks {
		if mark.Alias != "" {
			fmt.Printf("[%v] [%v] %v\n", index, mark.Alias, mark.Path)
			continue
		}
		fmt.Printf("[%v] %v\n", index, mark.Path)
	}
}

func (m *MarkCli) Add(args []string) {
	if len(args) > 1 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	alias := ""
	if len(args) == 1 {
		alias = args[0]
	}
	path, err := os.Getwd()
	m.handleError(err)
	marks, err := m.db.List()
	m.handleError(err)
	m.db.Clear()
	temp := marks
	m.db.Add(&Mark{
		Path:  path,
		Alias: alias,
	})
	for _, mark := range marks {
		err := m.db.Add(mark)
		if err != nil {
			m.db.Clear()
			for _, item := range temp {
				m.db.Add(item)
			}
			m.handleError(err)
		}
	}
}

func (m *MarkCli) Forward(args []string) {
	if len(args) != 1 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	cwd, err := os.Getwd()
	m.handleError(err)
	regex, err := regexp.Compile(args[0])
	m.handleError(err)

	var matches []struct {
		Name  string
		IsDir bool
	}
	filepath.WalkDir(cwd, func(path string, d fs.DirEntry, err error) error {
		if regex.MatchString(d.Name()) {
			matches = append(matches, struct {
				Name  string
				IsDir bool
			}{
				Name:  path,
				IsDir: d.IsDir(),
			})
		}
		return nil
	})

	if len(matches) == 0 {
		m.handleError(errors.New("no matches found"))
	}

	for index, path := range matches {
		fmt.Fprintf(os.Stderr, "[%v] %v\n", index, path.Name)
	}

	var choice int
	for {
		fmt.Fprint(os.Stderr, "Choose an option: ")
		_, err = fmt.Scanf("%d", &choice)
		if err == nil && choice >= 0 && choice < len(matches) {
			break
		}
		fmt.Fprintln(os.Stderr, "invalid index try again")
	}
	item := matches[choice]
	path := item.Name
	if !item.IsDir {
		arr := strings.Split(item.Name, "/")
		if len(arr) != 1 {
			path = strings.Join(arr[0:len(arr)-1], "/")
		}
	}
	fmt.Println(path)
}

func (m *MarkCli) Get(args []string) {
	if len(args) > 1 {
		m.handleError(errors.New("invalid number of arguments"))
	}
	var err error
	var mark *Mark
	index, err := strconv.Atoi(args[0])
	if err != nil {
		mark, err = m.db.GetByAlias(args[0])
		if err != nil {
			m.handleError(err)
			return
		}
	} else {
		mark, err = m.db.GetByIndex(index)
		if err != nil {
			m.handleError(err)
			return
		}
	}
	m.handleError(err)
	fmt.Println(mark.Path)
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

jump() {
	local readonly DEST=$(mark jump $1)
	if [[ ! -z $DEST ]]; then
		cd $DEST
	fi
}

forward() {
	local readonly DEST=$(mark forward $1)
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
	if err != nil {
		m.handleError(errors.New("invalid index specified"))
	}
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
		"jump":    func(args []string) { mark.Jump(args) },
		"forward": func(args []string) { mark.Forward(args) },
		"switch":  func(args []string) { mark.Switch(args) },
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
