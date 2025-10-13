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

## File Create:

output the content of the file.

`path/to/file`

```
...
file content
...
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

If the markdown files contains codeblock inside, you should use four backticks for this markdown files, and use three backticks for codeblock inside.

`file1.md`

````markdown
...

```python
...
```

...
````

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
