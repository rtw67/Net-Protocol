#!/bin/bash
# 创建一个空的二进制文件
touch binary_file
# 循环遍历数组中的每个元素
for decimal in 186 42 242 73 255 128 186 42 242 73 255 128 8 0 69 0 0 84 212 50 64 0 64 1 138 24 172 17 110 161 192 168 1 3 8 0 54 8 0 24 0 1 238 181 157 100 0 0 0 0 117 241 1 0 0 0 0 0 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
do
  # 将十进制数转换成十六进制数，并写入二进制文件
  printf "\\x$(printf '%x' $decimal)" >> binary_file
done
