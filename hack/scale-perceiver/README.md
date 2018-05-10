# ScalePerceiver

Intended usage: shove a whole bunch of images into Perceptor, structured so that there's between 1-10 versions per project.

Usage:

```
python main.py <perceptor_url> <number_of_scans>
```

# Notes

 - expects to be able to hit the Docker command line client in order to get a bunch of shas
 - currently uses some shell commands which probably wouldn't be available from within a container