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
|help|Displays help menu|
|add|Adds the current working directory to mark db (Default action)|
|back <index>|Prints out the number of directories back| 
|clear|Clears out the paths in mark db|
|delete <index>|Deletes out a path in mark db based on the index provided|
|get <index>|Get the path in mark db based on the index provided|
|list|List out all the marked paths by index|
|install|Prints out directions to create move and back commands in your .bashrc|


