import subprocess
from StringIO import StringIO
import json
import random
import sys
# import requests # argh, can't import


image_name = 'gcr.io/gke-verification/blackducksoftware/echoer'


csv = subprocess.Popen(
	"docker image ls --digests {}".format(image_name),
	shell=True,
	stdout=subprocess.PIPE).stdout.read()

def parse_sha(line):
	columns = line.split()
	sha = columns[2][7:]
	return sha

all_but_first_and_last_lines = csv.split('\n')[1:-1]
shas = map(parse_sha, all_but_first_and_last_lines)

def get_random_nums_totalling(total):
	nums = []
	running_total = 0
	while sum(nums) < total:
		next = random.randint(1, 10)
		if (next + running_total) > total:
			next = total - running_total
		running_total += next
		nums.append(next)
	return nums

def make_scan(sha, project, version, scan):
	return {
		'Name': image_name,
#		'PullSpec': '{}@sha256:{}'.format(image_name, sha),
		'Sha': sha,
		'Project': project,
		'Version': version,
		'Scan': scan
	}

def make_projects(total_scans):
	scans = []
	projects = []
	scan_count = 0
	version_counts = get_random_nums_totalling(total_scans)
	for (project_id, version_count) in enumerate(version_counts):
		project = "test-project-{}".format(project_id)
		versions = []
		for version_id in range(version_count):
			scan_index = scan_count % len(shas)
#			print project_id, version_id, scan_count, shas[scan_index]
			version = "proj-{}-version-{}".format(project_id, version_id)
			scan = "proj-{}-version-{}-scan-{}".format(project_id, version_id, scan_count)
			sha = shas[scan_index]
#			print "sha: ", sha, shas
			versions.append({'Name': version, 'Scan': {'Name': scan, 'Sha': sha, 'PullSpec': "{}@sha256:{}".format(image_name, sha)}})
			scans.append(make_scan(sha, project, version, scan))
			scan_count += 1
		projects.append({'Name': project, 'Versions': versions})
	return (projects, scans)

def send_request_to_perceptor(url, scan):
	full_url = "{}/image".format(url)
	command = """curl --header "Content-Type: application/json" -X POST --data '{}' {}""".format(json.dumps(scan), full_url)
	print command
	r = subprocess.Popen(
		command,
		shell=True,
		stdout=subprocess.PIPE).stdout.read()
# 	r = requests.post(url, data=scan)
	return r


perceptor_url, scan_count = sys.argv[1], int(sys.argv[2])

projects_and_scans = make_projects(scan_count)

for scan in projects_and_scans[1]:
	print json.dumps(scan, indent=2)
	print send_request_to_perceptor(perceptor_url, scan)

# print json.dumps(make_projects(scan_count), indent=2)