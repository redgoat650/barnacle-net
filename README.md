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