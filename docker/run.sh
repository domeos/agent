#!/bin/bash

./dockerConnector &
bash ./control start
bash ./control tail
