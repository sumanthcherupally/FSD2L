import csv
import networkx as nx
import numpy as np
import requests
import itertools
#from bitstring import BitArray
from random import *
import random
from operator import itemgetter
import math
import operator
import time
import sys
import re
from collections import OrderedDict
import ipaddress
from datetime import datetime, timedelta
import json 
import threading
import requests


BlockchainNodesIP = []

nodesFile = open("bc_nodes.txt", 'r')

for line in nodesFile:
	s = line.replace("\n","")
	BlockchainNodesIP.append(s)

print(BlockchainNodesIP)

url = ""

result=  'bgp-data-path.csv' 
#bw_details = 'BW_OF_LINKS.csv'
fp = open(result, 'a')
fp.write("Type, Prefix, S-AS, D-AS \n")

pfx = '10.0.0.0/8' ### initial prefix from which subnets for each node is derived ###########
net = ipaddress.ip_network(pfx)

N = int(sys.argv[1])  ## number of nodes in Internet AS graph
#P = int(sys.argv[2])  ### number of prefixes per node


G = nx.random_internet_as_graph(N)
#print (G.edges(), G.nodes())

prefix_graphs = {}


prefix_len = math.ceil(math.log(N,2))
#print (prefix_len)
subnets = list(net.subnets(prefixlen_diff=prefix_len))

node_prefixes = {}

for i in G.nodes():
	node_prefixes[i] = str(subnets[i])


########### assign transactions data #############################################
for i in node_prefixes:
	fp.write(str('assign')+ "," + str(node_prefixes[i]) + "," + str(65536) +  "," + str(i) + "," + "\n")
	PrefixAllocateMsg = {}
	PrefixAllocateMsg["Prefix"] = str(node_prefixes[i])
	PrefixAllocateMsg["Source"] = 65536
	PrefixAllocateMsg["DestinationAS"] = int(i)
	PrefixAllocateMsg["StartTime"] = 1
	PrefixAllocateMsg["EndTime"] = 1000
	Ip = BlockchainNodesIP[i % len(BlockchainNodesIP)]
	url = "http://"+Ip+":8989/"
	dat = json.loads(json.dumps(PrefixAllocateMsg))
	requests.post(url + "prefixAllocate", json = dat)

################################################################################

############ generate path transactions upto length 4 starting from each node ###############

def generate_data(origin,prefix, IpAddress):
	H = nx.DiGraph()
	#print (i, list(G.neighbors(s)))
	#visited = [False] * len(G.nodes())
	forwarded = {}
	for node in G.nodes():
		forwarded[node] = False
	queue = []
	queue.append((origin,0)) ### node and depth
	H.add_node(origin)
	#prefix = node_prefixes[origin]
	prefix_received_from_neighbors = {}
	for node in G.nodes():
		prefix_received_from_neighbors[node] = [] ### {1:[2,3]} store from which nodes u received prefix dont send it to them
	while queue:
		#print (queue)
		(s,d) = queue.pop(0)
		#print ('vertex,depth',s,d)
		if forwarded[s] == False:
			if (d <= 4):
				for i in list(G.neighbors(s)):
					if i not in list(set(prefix_received_from_neighbors[s])):
						H.add_edge(s,i)
						path = nx.shortest_path(H, source = origin, target = s) #### path from origin till node s ##
						fp.write(str('announce')+ "," + str(prefix) + "," + str(s) + "," + str(i) + ", " + str(path) + "," + "\n")
						print ('s, neighbor, prefix', s, i, path, prefix)
						# now create a json object and send it to the blockchain nodes
						PathAnnounceMsg = {}
						PathAnnounceMsg["Prefix"] = str(prefix)
						PathAnnounceMsg["SourceAS"] = int(s)
						PathAnnounceMsg["DestinationAS"] = int(i)
						PathAnnounceMsg["Path"] = path
						dat_ = json.loads(json.dumps(PathAnnounceMsg))
						url = "http://"+IpAddress+":8989/"
						requests.post(url + "pathAnnounce", json = dat_)
						time.sleep(1)
						queue.append((i,d+1))
						prefix_received_from_neighbors[i].append(s)
					#if (visited[i] == False):
					#visited[s] = True
			else:
				break
			forwarded[s] = True

threads = []


def main():
	#global node_prefixes
	for i in G.nodes():
		prefix = node_prefixes[i]
		IP = BlockchainNodesIP[i % len(BlockchainNodesIP)]
		t = threading.Thread(target=generate_data, args= (i,prefix, IP))
		# generate_data(i,prefix)
		t.start()
		threads.append(t)

	for t in threads:
		t.join()

	time.sleep(1)
	path = [1,43,73,12,65]
	fp.write(str('announce')+ "," + str("10.1.0.0/4") + "," + str(1) + "," + str(65) + ", " + str(path) + "," + "\n")
	print ('s, neighbor, prefix', 1, 65, path, "10.1.0.0/4")
	PathAnnounceMsg = {}
	PathAnnounceMsg["Prefix"] = str("10.1.0.0/4")
	PathAnnounceMsg["SourceAS"] = 1
	PathAnnounceMsg["DestinationAS"] = 65
	PathAnnounceMsg["Path"] = path
	dat_ = json.loads(json.dumps(PathAnnounceMsg))
	url = "http://"+IP+":8989/"
	requests.post(url + "pathAnnounce", json = dat_)

	time.sleep(1)
	fp.write(str('assign')+ "," + str(node_prefixes[9]) + "," + str(9) +  "," + str(5) + "," + "\n")
	PrefixAllocateMsg = {}
	PrefixAllocateMsg["Prefix"] = str(node_prefixes[9])
	PrefixAllocateMsg["Source"] = 9
	PrefixAllocateMsg["DestinationAS"] = 5
	PrefixAllocateMsg["StartTime"] = 1
	PrefixAllocateMsg["EndTime"] = 1000
	url = "http://"+IP+":8989/"
	dat = json.loads(json.dumps(PrefixAllocateMsg))
	requests.post(url + "prefixAllocate", json = dat)

	time.sleep(1)
	fp.write(str('assign')+ "," + str(node_prefixes[10]) + "," + str(6) +  "," + str(2) + "," + "\n")
	PrefixAllocateMsg = {}
	PrefixAllocateMsg["Prefix"] = str(node_prefixes[10])
	PrefixAllocateMsg["Source"] = 6
	PrefixAllocateMsg["DestinationAS"] = 2
	PrefixAllocateMsg["StartTime"] = 1
	PrefixAllocateMsg["EndTime"] = 1000
	url = "http://"+IP+":8989/"
	dat = json.loads(json.dumps(PrefixAllocateMsg))
	requests.post(url + "prefixAllocate", json = dat)


main()
fp.close()