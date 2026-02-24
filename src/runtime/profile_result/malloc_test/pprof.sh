#!/bin/bash
# cfgs=(nops_mask nodmb eager_calc default batch)
cfgs=(default batch)
port=18121
ids=()
for cfg in "${cfgs[@]}"
do
    go tool pprof -http=127.0.0.1:$port $cfg.out &
    ids+=($!)
    ((port+=1))
done
sleep 1
echo "Enter q to kill server ${ids[@]}.
Please wait before return code is echoed."

cleanup() {
    kill "${ids[@]}"
    echo "kill return code: $?"
}

while true; do
    # -n 1: 只读取一个字符
    # -s: 隐藏输入 (可选，这里不隐藏为了用户体验)
    # -r: 防止反斜杠被解释
    read -r -n 1 -p "Input (q to quit): " user_input
    
    # 打印换行符，使下一次提示符在新的一行
    echo
    
    # 检查输入是否是 'q' 或 'Q'
    if [[ "$user_input" == "q" || "$user_input" == "Q" ]]; then
        cleanup
        exit
    fi
done
