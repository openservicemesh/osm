Adding a REST service to OSM Mesh
---------------------------------
In case of OSM, REST service will bypass envoy. We add iptables rules to bypass it.
1. Modify init-iptables.sh
    Add a line like below :
        iptables -t nat -A PROXY_REDIRECT -p tcp --dport "22" -j ACCEPT # ssh port
    Use the new REST port # instead of 22, also modify the description.
    Add it along with other PROXY_REDIRECT rules. Maintain increasing order of port numbers.
    Similarly, add another inbound rule like below along with other PROXY_INBOUND rules
        iptables -t nat -A PROXY_INBOUND -p tcp --dport "49" -j RETURN  # tacacs
    change port # to the new REST port #.
2. Edit .env file to change the following line
        export CTR_TAG=osmlatest3
   Bump up the version to something like "osmlatest4" or whatever would be latest.
3. Run ws/build.sh
   this will build and push the latest OSM image to repo.
4. Goto controller workspace
   Modify cloud-scripts/osm/values.yaml with the new tag you created.
    image:
      registry: docker.dev.ws:5000
      pullPolicy: Always
      tag: osmlatest3                      <== use the new tag you created in step 2.
5. Deploy using build_osm_cloud.sh and test your changes.
   You can use "iptables -L -t nat" on the container to see if your rule shows up.
   (install iptables using apk add if iptables is not found)
6. Merge init-iptables.sh and .env files to osm repo.
   Don't forget this step.
7. Merge cloud-scripts/osm/values.yaml into controller repo.
   Don't forget this step.
