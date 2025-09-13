# mark: A cli for saving your current working directory

## Description
The mark command quickly saves your current directory to be used later as a jump point.
```
> mark
> cd ../../
> move
> pwd
```

## Commands

|command|description|
|-|-|
|add|Adds the current working directory to mark db (Default action)|
|back <index>|Prints out the number of directories back| 
|clear|Clears out the paths in mark db|
|delete <index>|Deletes out a path in mark db based on the index provided|
|forward <regex>|Looks foward for directories that match a regex|
|get <index>|Get the path in mark db based on the index provided|
|help|Displays help menu|
|install|Prints out directions to create move and back commands in your .bashrc|
|jump|Prints out the number of directories jumping forward from the beginning|
|list|List out all the marked paths by index|
|switch <source> <dest>|Switch stored paths by their index|
