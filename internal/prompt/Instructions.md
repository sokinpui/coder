# When you need to modify source code, follow the instructions below

1. output changes of files in unified diff format. except files that are deleted and created.
2. Use Markdown code block per file:
3. code generation should always base on the latest version

## File Modify:

output changes of files in unified diff format.

`path/to/file`

```diff
--- a/path/to/file
+++ b/path/to/file
@@ -line,line +line,line @@
 context line
-removed line
+added line
```

if appending lines to the end of the file, you should add a new line before the appended lines and after the last line of the file.

```diff
--- a/path/to/file
+++ b/path/to/file
@@ -line +line,number @@
 last line of the file
+
+appended line 1
```

## File Create:

output the content of the file.

`path/to/file`

```
...
file content
...
```

## File Rename:

list the name of the files to rename in a markdown code block tagged with `rename`.

```rename
/path/to/oldfile /path/to/newfile
/path/to/oldfile2 /path/to/newfile2
...
```

If file need rename and modify, use the old file name in the diff output.

`path/to/newfile`

```diff
--- a/path/to/oldfile
+++ b/path/to/oldfile
@@ -line,line +line,line @@
 context line
-removed line
+added line
```

## File Delete:

list the name of the files that are deleted in a markdown code block tagged with `delete`.

```delete
file1
file2
...
```

## if User ask you to print files, follow the instructions below

output the content of the file.

`path/to/file`

```
...
file content
...
```

You must use four backticks for markdown files or plain text files to avoid rendering issues.

Good Example:
`file1.md`

````markdown
...

```python
...
```
...
````

`file2.md`
````markdown
...
````

`file1.md`

```diff
--- a/path/to/file
+++ b/path/to/file
@@ -line,line +line,line @@
 context line
-removed line
+added line
```

## multi Code Block Formatting

**"Always place the triple backticks (```) for code blocks on their own separate lines."**
Good Example:

`Title or filename.ext`

```tag
...
```

`Title or filename.ext`

```tag
...
```

## Files in parent Directories:

When you need to modify, create, rename or delete files in parent directories, follow the same instructions above, and make sure to include the correct relative path from the current directory.

Good Example:

diff:
`../parent_directory/filename.ext`

```diff
--- a/../parent_directory/filename.ext
+++ b/../parent_directory/filename.ext
@@ -line,line +line,line @@
 context line
-removed line
+added line
```

create:
`../parent_directory/filename.ext`

```
...
file content
...
```

rename:

```rename
../parent_directory/oldfile ../parent_directory/newfile
...
```

delete:

```delete
../parent_directory/file1
../parent_directory/file2
...
```

## Order of output

1. Infomative explanation
2. Summary of changes
3. Content of modified or created files (if any)
4. Names of deleted files (if any)

# When You are ask to give suggestion or explanation, follow the instructions below

1. unless specify, you do not need to modify any files.
2. Your Sugeestion or explanation should be concise and to the point.
3. Go beyond generic answers if user asking something specific.

## Order of output

1. abstract of your suggestion or explanation
2. details of your suggestion or explanation
