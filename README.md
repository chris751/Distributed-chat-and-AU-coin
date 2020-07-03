# Distributed-chat-and-currency-2018
A distributed chat, where clients relay messages to eachother, aswell as transfering the cryptocurrency 'AU Coin' to eachother without any centralized entity. Project made in 2018 for the course Distributed Systems and Security. 

# System Test

The system can obtain agreement between three peers. The code was tested by running the peers asdescribed above. After a peers completes a transaction, it is distributed in total order by the winnerof the lottery. The lottery winner is agreed upon by all the peers in the system. If a peer decides tobe malicious and try to transfer to himself this will be rejected by the system. A transaction thata given peer will do is signed and verified by other peers on the network. If any alterations is doneby and adversary to the signature or the transaction the transaction will be ignored. When thelottery winner adds a block to the network the block is only added to the blockchain if the messageis verified by the receiver.

# How to Run 
Run client.go. enter some random input at the first peer. Thenext peer should connect to the initial peer. The IP is hardcoded to be localhost and the port isprinted from peer 1. Peer 2 should now be connected to p1. Do the same with p3. The system innow done initiating, and transactions can now be entered in the terminal. Enter the receiver(pk),followed by the amount. Unfortunately, due to random crashes, the system currently only workswith 3 peers.
