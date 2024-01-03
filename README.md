# apple2disk

Tools for managing apple II Disk II contents.

## License
All tools released under GNU GPL v3.0 : See [LICENSE](./LICENSE)

## floppy_disk_image_file_to_serial_install
This program takes a ProDos logical order disk image file (\*.PO) and a track number as input and outputs to stdout a series of commands to the apple \]\[ system monitor to fill memory buffers, load a machine language program, and execute that program. The result is that one track of data from the disk image file is written to the Disk II floppy disk.

The program is written in go language (version 1.14), and must be compiled before running. If go and GNU Make are available on your system, you can use the Makefile to compile the program like this:
```
mkdir bin
make all
```

The output from this program must be sent over a serial connection to an appropriately readied apple \]\[ computer with a disk drive and inserted floppy disk.
- The apple must have been booted into DOS so that the RWTS subroutine of DOS is loaded into memory.
- The apple must be connected to your transmitting computed with a serial connection. An example of this is to use the 5 pin DIN connector on the rear right side of the apple II c (serial port #2), wired appropriately to an RS232 serial port on the transmitting computer. The apple port could be initialized with the basic command "IN#2", followed by commands to set 2400 baud (CTRL-A B 10) and 7 data bits plus 2 stop bits frame (CTRL-A D 5).
- The apple must be put into the monitor routine (at the basic prompt this could be done with CALL -151)
- Then the transmission could be made over the serial port and the commands would be interpreted by the monitor.

Example execution with arguments:

```
% bin/floppy_disk_image_file_to_serial_install "na.boot_D1_S2.PO" 0 > "d1s1t0.txt"
```

This would write the commands for loading and writing track 0 of disk image "na.boot_D1_S1.PO" into a file called d1s1t0.txt. That track could then be sent over the serial connection using a file transfer utility.
To write a complete disk side, 35 such track files would need to be transmitted.
