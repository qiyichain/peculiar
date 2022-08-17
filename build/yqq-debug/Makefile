init:
	../bin/geth --datadir=data init ./genesis.json

clean:
	rm -rf ./data/geth

start:
	../bin/geth --datadir data --ipcdisable --syncmode full --http --allow-insecure-unlock --http.addr "0.0.0.0" --unlock 0xb45b3b1aa3532477d8c915b3056bff2388deaa3c --password password.txt --mine