// Patch Witcher 3 for ultrawide cutscenes with no black bars
// Based on:
// https://vulkk.com/2021/06/27/how-to-fix-witcher-3-ultrawide-cutscenes-no-black-bars/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	filename  = "witcher3.exe"
	targetDir = "x64_dx12"

	// Common Name	Formatted Value
	// 5:04		00 00 A0 3F
	// 4:03		AB AA AA 3F
	// 3:02		00 00 C0 3F
	// 16:10	CD CC CC 3F
	// 15:09	55 55 D5 3F
	// 16:09	39 8E E3 3F
	// 1.85:1	CD CC EC 3F
	// 2.39:1	C3 F5 18 40
	// 2.76:1	D7 A3 30 40
	// 3x5:4	00 00 70 40
	// 3x4:3	00 00 80 40
	// 3x16:10	9A 99 99 40
	// 3x15:9	00 00 A0 40
	// 3x16:9	AB AA AA 40
	// 21:9 (2560x1080)	26 B4 17 40
	// 21:9 (3440x1440)	8E E3 18 40
	options = map[string][]byte{
		"2560x1080": {0x26, 0xB4, 0x17, 0x40}, // 26 B4 17 40
		"3440x1440": {0x8E, 0xE3, 0x18, 0x40}, // 8E E3 18 40
		"3840x1600": {0x9A, 0x99, 0x19, 0x40}, // 9A 99 19 40
		"5120x1440": {0x39, 0x8E, 0x63, 0x40}, // 39 8E 63 40
		"5120x2160": {0x26, 0xB4, 0x17, 0x40}, // 26 B4 17 40
		"6880x2880": {0x8E, 0xE3, 0x18, 0x40}, // 8E E3 18 40
	}

	hexValue = []byte{0x39, 0x8e, 0xe3, 0x3f} // "42 08 2b ca" - 16:09
)

func showUsage() {
	usage := `Usage: witcher-3-ultrawide-fix.exe "C:\Games\GOG\The Witcher 3 Wild Hunt" 3440x1440

	Use quotes " around the path
Options:
  --help     Show this message
`
	fmt.Fprint(os.Stderr, usage)
}

func main() {
	var help bool
	flag.BoolVar(&help, "help", false, "Show usage message")
	flag.Parse()
	if help {
		showUsage()
		os.Exit(0)
	}
	if len(os.Args) != 3 {
		showUsage()
		os.Exit(0)
	}
	directory := os.Args[1]
	resolution := os.Args[2]

	if stat, err := os.Stat(directory); err != nil || !stat.IsDir() {
		fmt.Printf("%s does not exists\n", directory)
		os.Exit(1)
	}
	if _, ok := options[resolution]; !ok {
		fmt.Printf("%s is not a valid option, choose one:\n", resolution)
		for key := range options {
			fmt.Println(key)
		}
		os.Exit(1)
	}
	file, err := FindWitcher(directory)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	dst, err := os.Create(filepath.Join(directory, "witcher3_backup.exe"))
	if err != nil {
		fmt.Printf("could not create a backup: %s\n", err)
		os.Exit(1)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		fmt.Printf("could not do backup: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("created a backup at %s\n", dst.Name())

	result, err := PatchWitcher(file, options[resolution])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(result)
	os.Exit(0)
}

func FindWitcher(rootDir string) (*os.File, error) {
	var executable string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == targetDir {
			err := filepath.Walk(path, func(innerPath string, innerInfo os.FileInfo, innerErr error) error {
				if innerErr != nil {
					return innerErr
				}
				if !innerInfo.IsDir() && innerInfo.Name() == filename {
					executable = innerPath
					return filepath.SkipDir
				}
				return nil
			})
			if err != nil {
				return err
			}
			if len(executable) > 1 {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(executable) == 0 {
		return nil, fmt.Errorf("file %s not found in %s", filename, rootDir)
	}
	return os.OpenFile(executable, os.O_RDWR, 0644)
}

func PatchWitcher(file *os.File, newHexValue []byte) (string, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	count := 0
	for i := 0; i < len(content)-3; i++ {
		if content[i] == hexValue[0] && content[i+1] == hexValue[1] &&
			content[i+2] == hexValue[2] && content[i+3] == hexValue[3] {
			copy(content[i:i+4], newHexValue)
			count = count + 1
		}
	}

	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	if _, err := file.Write(content); err != nil {
		return "", err
	}
	if count == 0 {
		return "looks like witcher 3 is already patched", nil
	}
	return fmt.Sprintf("updated %d times, expected 3\n", count), nil
}
