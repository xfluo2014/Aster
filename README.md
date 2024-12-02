# Aster: A Blockchain Architecture for Lightweight Nodes  

Aster is a next-generation blockchain architecture designed to optimize scalability and resource efficiency by integrating lightweight nodes. This system reduces the computational and storage burden on full nodes while maintaining decentralization and performance.

---

## **Key Features**  

### 1. **Node Roles and Responsibilities**  
- **Proposer Nodes**  
  - Manage transaction pools, schedule transactions, and package validated transactions into blocks for broadcast.  
  - Delegate tasks to lightweight **follower nodes**, reducing resource consumption on the proposer side.  

- **Follower Nodes**  
  - Maintain a specific branch of the global state (e.g., a subtree of the Merkle Patricia Trie).  
  - Execute and validate transactions relevant to their assigned state, returning Merkle proofs to the proposer.  
  - Designed for devices with limited computational and storage capabilities.  

---

### 2. **Task Offloading and Incentives**  
- **Proposer Node Recruitment**  
  - Proposers recruit followers using decentralized node discovery protocols.  
  - Relationships are formalized through smart contracts, which manage follower registration, task assignments, and reward distribution.  

- **Incentive Distribution**  
  - Smart contracts ensure automatic and transparent payment of rewards to followers, encouraging participation and reliability.  

---

### 3. **Efficient Cross-Node Transactions**  
- Transactions spanning multiple followers are handled via **Merkle proofs**.  
- Proposers collect and verify these proofs, ensuring transaction correctness without excessive communication overhead.  

---

## **Advantages**  

### **Resource Optimization**  
- Proposer nodes offload computational and storage tasks to followers, reducing resource strain.  
- Follower nodes store only relevant state segments, minimizing memory and storage usage.  

### **Scalability**  
- Proposers can dynamically recruit followers as the network grows, enabling efficient scaling to support high transaction volumes.  

### **Fault Tolerance**  
- Aster includes timeout mechanisms and task reassignment strategies to ensure stability in the event of node failure.  

### **Simplified Incentives**  
- Rewards are distributed in a mining pool-like manner, ensuring fairness and minimizing overhead.  

---

## **Getting Started**  

### **Clone the Repository**  
```bash  
git clone https://github.com/example/aster-blockchain.git  
