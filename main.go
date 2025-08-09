package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// Global flag variables. Are switched
// automatically in the flag parsing process
// (func main)
var FLAG_VERBOSE = false
var FLAG_IGNORE_NULL = false
var FLAG_EXCLUDE_ERRORS = false

// Glogal variables for the entry directory
// and ignored elements
var SCAN_DIR string
var IGNORE_DIRS = []string{}
var IGNORE_EXT = []string{}

// Struct for keeping track of amount of lines
// for all files with fType format
type FileTypeLineCount struct {
	fType     string // File type extension
	lineCount int    // Total amount of lines in files with the fType extension
}

// Struct for storing a flag pair:
// A string representing a flag and
// an address of a variable which is
// to be changed if the flag is present
type FlagPair struct {
	flag     string // String representing the flag
	variable *bool  // Address of the variable
}

// Flags array for automation of switch
// flags
var FLAGS = []FlagPair{
	{"-v", &FLAG_VERBOSE},
	{"-n", &FLAG_IGNORE_NULL},
	{"-r", &FLAG_EXCLUDE_ERRORS},
}

func logI(args ...string) {
	if FLAG_VERBOSE {
		color.New(color.FgBlue, color.Bold).Print(args[0])
		for _, v := range args[1:] {
			fmt.Print(v)
		}
		fmt.Println()
	}
}

func logE(args ...string) {
	if FLAG_VERBOSE {
		color.New(color.FgRed, color.Bold).Print(args[0])
		for _, v := range args[1:] {
			fmt.Print(v)
		}
		fmt.Println()
	}
}

// Return true if a dir is to be ignored
// (If the user has excluded it from the scan).
// Here, any valid path is considered a dir.
func dirIsIgnored(dir string) bool {
	for _, v := range IGNORE_DIRS {
		if dir == v {
			return true
		}
	}
	return false
}

// dirIsIgnored but for a file extension.
// Again, set by user. The name of the
// extension should come WITHOUT a dot.
func fileExtIsIgnored(ext string) bool {
	for _, v := range IGNORE_EXT {
		if ext == v {
			return true
		}
	}
	return false
}

// Pare a file path to determine it's extension.
// Returns an extension WITHOUT a dot.
func getFileNameExtension(name string) string {
	lstInd := strings.LastIndex(name, ".")
	lstSlash := strings.LastIndex(name, "/")
	if lstInd > lstSlash {
		return name[lstInd+1:]
	} else {
		return "NULL"
	}
}

// Main utility logic. The file search is perfomed via recursive search,
// where each scannable directory is an individual node. If the scanner
// encounters a new directory, it adds it to the queue, which is then passed
// as an argument to the next node to scan. The scan process finishes once the
// queue is empty.
func node(dir string, dir_queue []string, file_data []FileTypeLineCount, file_count int, skipped_files []string) {
	// Assign passed arguments to local variables,
	// a readability boost for sure
	current_sf := skipped_files
	current_fc := file_count
	current_fd := file_data
	current_queue := dir_queue

	// Read the contents of the dir, abort scan if
	// unaccessable. The 'dir' is basically the first
	// element of the 'dir_queue', don't really
	// know why I made it that way
	content, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Iterate through the contents of the
	// scanned dir
	for _, i := range content {
		if i.IsDir() {
			// If an element is a dir and is not set to be
			// ignored, append it to the queue. Ignore otherwise
			if !dirIsIgnored(dir + "/" + i.Name()) {
				current_queue = append(current_queue, dir+"/"+i.Name())
				logI("added ", i.Name(), " to the queue")
			} else {
				logI("ignoring ", i.Name())
				current_sf = append(current_sf, dir+"/"+i.Name()+"/")
			}
		} else {
			// Get a relative path to the current
			// scaned file
			file_path := dir + "/" + i.Name()
			// Scan the file and add it's LOC to
			// the element of the 'file_data' corresponding
			// to it's type if the file is not set to be ignored.
			// Ignore otherwise
			if dirIsIgnored(file_path) {
				logI("ignoring ", i.Name())
				current_sf = append(current_sf, file_path)
			} else {
				logI("processing ", i.Name())
				current_fc++
				// Open file and check for errors.
				file, err := os.Open(file_path)
				if err != nil {
					fmt.Println("error opening '"+file_path+"':", err)
					current_sf = append(current_sf, file_path+"]ERR")
				}

				scanner := bufio.NewScanner(file)
				line_count := 0
				// Read file's lines of code (basically don't add
				// a line if it's empty)
				for scanner.Scan() {
					if len(scanner.Bytes()) > 0 {
						line_count++
					}
				}
				// Probably should log the reading error despite
				// the verbose flag being off, not an issue for now.
				if err := scanner.Err(); err != nil {
					logE("error", " reading '", file_path, "': ", err.Error())

					if FLAG_EXCLUDE_ERRORS {
						logI("ignoring ", file_path, "'s lines (user defined)")
						current_sf = append(current_sf, file_path+"]ERR")
						continue
					}
				}

				// Don't know why it's here, probably can be useful
				// if I commented it out instead of deleting though
				// logI("adding ", file_path, "'s lines to result")

				// Close the file and parse it's extension
				file.Close()
				file_ext := getFileNameExtension(file_path)
				file_count_added := false

				logI("'"+file_path+"'", " has ", strconv.Itoa(line_count), " lines of code; file extension: ", file_ext)

				// Add the file type's lines to 'file_data'
				// if an element corresponding to the type
				// already exists. Otherwise, create a new
				// element with some rules
				for i, v := range current_fd {
					if file_ext == v.fType {
						current_fd[i].lineCount += line_count
						file_count_added = true
					}
				}
				if !file_count_added {
					if file_ext == "NULL" {
						// Ignore the file if it doesn't have
						// an extension and the user has set a '-n'
						// flag on (see manual), add a NULL type
						// file otherwise
						if FLAG_IGNORE_NULL {
							logI("ignoring ", file_path, " (NULL extension)")
							current_sf = append(current_sf, file_path)
						} else {
							logI("adding", " new file type '", file_ext, "' with", strconv.Itoa(line_count), " lines of code")
							current_fd = append(current_fd, FileTypeLineCount{fType: file_ext, lineCount: line_count})
						}
					} else {
						// Ignore the file if it's extension was set
						// to be ignored by user. Add the file otherwise
						if fileExtIsIgnored(file_ext) {
							logI("ignoring ", file_path)
							current_sf = append(current_sf, file_path)
						} else {
							logI("adding", " new file type '", file_ext, "' with ", strconv.Itoa(line_count), " lines of code")
							current_fd = append(current_fd, FileTypeLineCount{fType: file_ext, lineCount: line_count})
						}
					}
				}
			}
		}
	}
	// The recursive part
	if len(current_queue) > 0 {
		// Start a new node if the queue is still there,
		// Scan the single remaining dir if not
		if len(current_queue) > 1 {
			node(current_queue[0], current_queue[1:], current_fd, current_fc, current_sf)
		} else {
			node(current_queue[0], nil, current_fd, current_fc, current_sf)
		}
	} else {
		// The scan is finished, the queue is empty,
		// all the file data is now stored in the arguments
		// to the last called node, so we can just call the
		// result screen from here and call it a day
		logI("queue empty", "; all files processed")
		result(current_fd, current_fc, current_sf)
		return
	}
}

// A really fancy way to display the result of a scan.
func result(file_data []FileTypeLineCount, files_count int, skipped_files []string) {
	// Never thought a terminal output design would take more
	// time and effort than recursive scanning a directory

	// Flags, will come in handy later
	null_type_present := false
	errors_present := false

	// Used for a cool equal spacing between a file
	// path and it's status (skipped files section)
	longest_skipped_file_len := -1

	// Colors
	white := color.New(color.FgWhite, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)

	// Element specific colors
	dir_name := color.New(color.FgHiCyan, color.BgBlack, color.Bold, color.Italic)
	null := color.New(color.FgRed, color.BgBlack, color.Bold, color.Underline)
	line_num := color.New(color.FgGreen, color.Bold)
	perc := color.New(color.FgHiMagenta, color.Bold)
	info := color.New(color.FgGreen, color.Bold, color.Italic)
	// info_null := color.New(color.FgRed, color.BgBlack, color.Bold, color.Italic)
	err := color.New(color.FgRed, color.Bold, color.Italic)

	// Count up total lines of code
	total_lines := 0
	for _, v := range file_data {
		total_lines += v.lineCount
	}

	white.Print("CBS REPORT ON '")
	dir_name.Print(SCAN_DIR)
	white.Println("':")

	// File listing cycle
	for _, v := range file_data {
		percentage := float64(v.lineCount) / float64(total_lines) * 100
		white.Print(" .")
		if v.fType == "NULL" {
			null.Print(v.fType)
			null_type_present = true
		} else {
			blue.Print(v.fType)
		}

		fmt.Print("\tfiles: ")
		line_num.Print("\t", v.lineCount, "\t")
		fmt.Print(" lines of code ")
		white.Print("[~")
		if percentage < 10 {
			perc.Print("0" + strconv.FormatFloat(percentage, 'f', 2, 64))
		} else {
			perc.Print(strconv.FormatFloat(percentage, 'f', 2, 64))
		}

		white.Println("%]")
	}

	// Get the spacing for the skipped files section
	for _, v := range skipped_files {
		if len(v) > longest_skipped_file_len {
			longest_skipped_file_len = len(v)
		}
	}

	read_err := []string{}
	for i, v := range skipped_files {
		// ']ERR' is a simple flag made up by me.
		// If the flag was appended to a path containing
		// the skipped file, that means an error occured
		// while reding the file, therefore it was skipped.

		// read_err will contain the paths to these files
		// without the flag, so they can later be printed
		// separatly without the need to remove the flag
		// while printing
		if v[len(v)-4:] == "]ERR" {
			read_err = append(read_err, v[:len(v)-4])
			skipped_files = append(skipped_files[:i], skipped_files[i+1:]...)
		}
	}

	if len(skipped_files) > 0 {
		// Skipped files and directories
		yellow.Println("\nSkipped files/directories: ")

		// First, print all the files that caused an error
		// while reading them
		for _, v := range read_err {
			err.Print(" " + v)
			fmt.Print(strings.Repeat(" ", longest_skipped_file_len-len(v)+1))
			err.Println("[READING ERROR]")
			errors_present = true
		}
		// Then print the excluded files and directories
		for _, v := range skipped_files {
			white.Print(" " + v)
			fmt.Print(strings.Repeat(" ", longest_skipped_file_len-len(v)+1))
			info.Println("[EXCLUDED BY USER]")
		}
	}

	// Errors info. The if statement pretty much
	// explains it
	if errors_present {
		red.Print("\nWarning! ")
		white.Print("CBS could not read some elements at '")
		dir_name.Print(SCAN_DIR)
		white.Println("'. Please,")
		white.Println("check accessibility and correctness of the elements marked with")
		white.Print("'")
		err.Print("[READING ERROR]")
		white.Print("' in the ")
		yellow.Print("Skipped files/directories")
		white.Println(" section.")
	}

	// Same with the NULL type
	if null_type_present {
		red.Print("\nWarning! ")
		white.Println("Some files scanned by CBS don't have an extension.")
		white.Println("This usually leads to incorrect line count in case the file is an executable.")
		white.Print("To ignore files without an extension use '")
		blue.Print("-n")
		white.Println("' when scanning.")
	}

	// Sum of the lines of code and the amount of
	// files containing them
	white.Print("\nTotal ")
	line_num.Print(total_lines)
	white.Print(" lines of code spread across ")
	blue.Print(files_count)
	white.Println(" files.")
}

// Print a quick usage guide to the terminal
func help() {
	// No point in commenting, just
	// colors and formatting and printing here

	header := color.New(color.FgBlue, color.Bold, color.Underline)
	parag := color.New(color.FgWhite)
	flag := color.New(color.FgCyan, color.Bold)
	cbs := color.New(color.FgMagenta, color.Bold, color.Italic)
	def := color.New(color.FgCyan, color.Bold)
	arg := color.New(color.BgBlack, color.FgGreen, color.Bold)
	note := color.New(color.FgWhite, color.Bold, color.Underline)

	cbs.Print("CBS ")
	def.Println("(Code Base Scanner)")
	parag.Println("\n - A simple command-line software to analyze the code base's size")
	parag.Print(" in lines of code. Each file extension's lines are counted separatly.\n\n")

	header.Print("USAGE\n\n")
	parag.Println(" Info: Specify a path to a directory to be scanned as the parameter")
	parag.Print(" followed by flags if necessary.\n\n")
	parag.Print(" Synopsis: ")
	cbs.Print("cbs ")
	arg.Print("[path]")
	fmt.Print(" ")
	arg.Print("[flags]")
	fmt.Print(" ")
	arg.Println("...")

	header.Print("\nFLAGS\n\n")
	flag.Print(" -v")
	parag.Println(" (VERBOSE) - basically a debug mode, logs almost every core")
	parag.Println("    operation, you won't really need this unless you're interested")
	parag.Print("    in understanding the method of gathering data\n\n")
	flag.Print(" -n")
	parag.Println(" (NULL) - ignore the files that don't have an extension. This")
	parag.Println("     is quite useful, since the file with no extension is most likely")
	parag.Println("     a binary or executable, in wich case the amount of lines will be")
	parag.Print("     invalid and misleading. It is recommended to use the -n flag by default.\n\n")
	flag.Print(" -r")
	parag.Println(" (READ ERROR) - ignore files with reading errors occuring in runtime. The amount")
	parag.Println("     of lines in a corrupted file won't affect the result. It is recommended to")
	parag.Print("     use the -r flag by default.\n\n")
	flag.Print(" -e ")
	arg.Print("[PATH]")
	parag.Print(" (EXCLUDE) - ignore ")
	arg.Print("[PATH]")
	parag.Print(" (directory or file) when scanning. ")
	note.Println("PLEASE")
	fmt.Print("     ")
	note.Print("NOTE, that")
	parag.Println(" the elements' path should not be described relative to the target")
	parag.Println("     directory. That means, if you are scanning '../dir', the excluded directory")
	parag.Println("     dir/dir2 would be '../dir/dir2', not 'dir/dir2'. Absence of a directory that")
	parag.Print("     is to beignored will not be considered an error by CBS or affect the scan result.\n\n")
	flag.Print(" -t ")
	arg.Print("[EXTENSION]")
	parag.Print(" (TYPE) - ignore every file with the ")
	arg.Print("[EXTENSION]")
	parag.Print(" file type.\n\n")
}

func main() {
	// Get the arguments, the first argument
	// is the command, won't need it
	args := os.Args[1:]

	// Print the manual if no arguments were passed
	if len(args) <= 0 {

		help()
		return
	}

	found_flag := false
	received_path := false
	skip_iter := false
	base_dir := "."

	// A flag parsing cycle, the swicth flag parsing
	// is automated here, the argument flags need to
	// be implemented by hand, but the algorithm is
	// pretty much the same
	if len(args) >= 1 {
		for i, v := range args {
			if skip_iter {
				skip_iter = false
				continue
			}
			found_flag = false
			for _, j := range FLAGS {
				if v == j.flag {
					*j.variable = true
					found_flag = true
				}
			}
			if v == "-e" {
				if len(args) < i+2 {
					fmt.Println("expected argument after '-e': directory name")
					return
				} else {
					IGNORE_DIRS = append(IGNORE_DIRS, args[i+1])
					found_flag = true
					skip_iter = true
				}
			}
			if v == "-t" {
				if len(args) < i+2 {
					fmt.Println("expected argument after '-t': file extesnion")
					return
				} else {
					IGNORE_EXT = append(IGNORE_EXT, args[i+1])
					found_flag = true
					skip_iter = true
				}
			}
			if !found_flag {
				if !received_path {
					base_dir = v
					received_path = true
				} else {
					fmt.Println("unexpected argument: '"+v+"' at argument position", strconv.Itoa(i+1)+"; the path has already been parsed as '"+base_dir+"'")
					return
				}
			}
		}
	}

	logI("starting CBS", " on ", base_dir)

	// Remove a '/' symbol on the entry directory
	// if present, things get ugly with it
	if base_dir[len(base_dir)-1] == '/' {
		base_dir = base_dir[:len(base_dir)-1]
	}
	// Set a global entry directory to the parsed
	// directory and start a node on it
	SCAN_DIR = base_dir
	node(base_dir, []string{}, []FileTypeLineCount{}, 0, []string{})
}
