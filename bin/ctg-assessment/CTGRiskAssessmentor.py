#!/usr/bin/python

import sys
import csv
import random

# $1: HR
# $2: UC
# $3: Settings
# $4: BPM

def main():
    hr, uc, params, bpm = sys.argv[1:]

    print("Data Name: {0} {1} {2} {3}".format(hr, uc, params, bpm))

    with open(hr, "r") as f:
        r = csv.reader(f)
        r.next()
        data = [row for row in r]

        # 3 Accelerations, 3 baselines(2 risks)
        events = sorted([random.randint(0, len(data)) for i in range(6)])

        for i, e in enumerate(events):
            d = data[e]

            begin = int(d[0])
            end = begin + 5000

            if i in (0, 1, 3):
                print("{0} - {1} Acceleration".format(begin, end))
            elif i in (2, 5):
                print("{0} - {1} Baseline-NORMAL {2}".format(begin, end, d[1]))
                print("{0} - {1} Risk {2}".format(begin, end, d[1]))
            else:
                print("{0} - {1} Baseline-NORMAL {2}".format(begin, end, d[1]))

    print("Data End")

if __name__ == "__main__":
    main()