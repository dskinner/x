#!/usr/bin/env bash

cp ./forward.bash ~/.forward.bash

if [ $(grep -c forward.bash ~/.bashrc) -eq 0 ]; then
    echo 'updating ~/.bashrc'
    echo '' >> ~/.bashrc
    echo '[ -f ~/.forward.bash ] && source ~/.forward.bash' >> ~/.bashrc
fi

