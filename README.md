Developing a distributed in-memory data store that you can scale read/writes onto many processes or VMs & be able to monitor its performance in Grafana

I intend to do the work on it from time to time while interning this Fall & combining it with my academic studies at my uni. The code here is not perfect; I adapt it as we go

What we have so far:
From having a single read/write node we expand our architecture to maintain a single master for write and multiple read replicas to ensure horizontal scalability for reads. This simple architecture ensures eventual consistency by utilizing asynchronous writes to all replicas that are being registered in-memory in the leader's memory.

Right now, we do not support:

leader election. So, master is our SPOF for writes and reads membership
no durable metadata for membership. If master fails, we effectively lose all our memberships
no health checks. We might send writes to dead nodes
we lack persistence that is needed if processes crash
no configurable consistency like acks or quorum
no retry or queue for failed replication attempts
no sharding to handle to scale writes horizontally
no proxy for client read routing or future writes
no consistent hashing
no monitoring as of now
