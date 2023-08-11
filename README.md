# barnacle-net

https://www.geekyhacker.com/configure-ssh-key-based-authentication-on-raspberry-pi/


ssh-keygen -t rsa

cat ~/.ssh/frames_rsa.pub | ssh redgoat@frame1.local "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"


https://stackoverflow.com/questions/18683092/how-to-run-ssh-add-on-windows


>ssh-agent

>ssh-add C:\Users\nwrig\.ssh\frames_rsa

>ssh redgoat@frame1.local


cat > ~/.ssh/config << EOF
Host f1 frame1.local
        HostName frame1.local
        IdentityFile ~/.ssh/frames_rsa
        User redgoat
EOF





https://docs.docker.com/engine/swarm/swarm-tutorial/create-swarm/

MANAGER_IP=192.168.1.50


 docker swarm init --advertise-addr $MANAGER_IP
<!-- Swarm initialized: current node (u7xdqf13xx8qw97h624j8tocj) is now a manager.

To add a worker to this swarm, run the following command:

    docker swarm join --token SWMTKN-1-5g9v45smbr1fbyi4r2uaxbpp03jok66ytpbe40uqshg09zelcj-0t4ihcekuapry391ftpbsd6dl 192.168.1.50:2377

To add a manager to this swarm, run 'docker swarm join-token manager' and follow the instructions. -->


docker info

docker node ls