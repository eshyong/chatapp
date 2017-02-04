#!/usr/bin/env python3

import random
import string

hash_key = ''.join([random.choice(string.hexdigits) for i in range(0, 64)])
block_key = ''.join([random.choice(string.hexdigits) for i in range(0, 32)])
print('Hash key: ' + hash_key)
print('Block key: ' + block_key)
