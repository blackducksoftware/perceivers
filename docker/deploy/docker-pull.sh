#!/bin/bash

uniqueImages=(centos alpine nginx busybox redis ubuntu mongo memcached mysql postgres
node registry golang hello-world php mariadb elasticsearch docker wordpress rabbitmq
haproxy ruby python openjdk logstash traefik debian tomcat influxdb java
swarm jenkins kibana maven ghost nextcloud cassandra telegraf kong nats
vault drupal fedora owncloud jruby sonarqube sentry solr gradle perl
rethinkdb neo4j percona groovy amazonlinux rocket.chat buildpack-deps chronograf redmine jetty
erlang couchdb pypy flink iojs couchbase zookeeper joomla django mono
piwik eclipse-mosquitto ubuntu-debootstrap bash crate nats-streaming elixir arangodb kapacitor tomee
haxe opensuse websphere-liberty adminer oraclelinux gcc orientdb rails mongo-express odoo
neurodebian ros xwiki clojure irssi ibmjava aerospike notary rust composer backdrop swift php-zendserver
r-base julia celery nuxeo docker-dev znc gazebo bonita cirros haskell plone hylang rapidoid geonetwork
eggdrop storm rakudo-star convertigo spiped hello-seattle mediawiki ubuntu-upstart lightstreamer fsharp
swipl glassfish thrift mageia photon open-liberty hola-mundo crux teamspeak sourcemage silverpeas
hipache scratch clearlinux si euleros clefos matomo)

imageCount=0
projectCount=0
hubHost=$1

if [[ $1 == "" ]] ;
then
	echo "Black Duck Hub host is required!"
	exit 1
fi

processImage() {
	projectName="$1"
	url="$2"
	randomNumber=$(($RANDOM % 10 + 1))
	for version in $(curl -s $url | jq '.[]' | tail -n +4 | jq '.[].name' | sed -e 's/^"//' -e 's/"$//' | head -n +$randomNumber)
	do
		image=$projectName:$version
		echo "Projectname: $projectName, Version: $version, Imagename: $image"
		BD_HUB_PASSWORD=blackduck ./scan.cli-4.7.0/bin/scan.docker.sh --image "$image" --username sysadmin --project "$projectName" --release "$version" --host $hubHost --port 443 --scheme https --insecure --no-inspect
		docker rmi $image
	done

	let imageCount++

	if [ $(($imageCount%8)) == 0 ] ;
	then
		sleep 600
	fi
}

for uniqueImage in "${uniqueImages[@]}";
do
	echo "Repo: $uniqueImage"
	for imageName in $(docker search $uniqueImage | cut -d ' ' -f 1 | tail -n +2)
	do
		if [[ "$imageName" == *\/* ]]
		then
			url="https://hub.docker.com/v2/repositories/$imageName/tags/"
			processImage "$imageName" "$url"
		else
			url="https://registry.hub.docker.com/v2/repositories/library/$imageName/tags/"
			processImage "$imageName" "$url"
		fi
		let projectCount++
		echo "project count: $projectCount"
		if [ $projectCount -gt 1000 ] ;
		then
			echo "Done completing $imageCount images"
			echo "Done completing $projectCount projects"
			exit 0
		fi
	done
done

echo "Total image count: $imageCount"
echo "Total project count: $projectCount"
