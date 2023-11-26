test:
	guile --r7rs -L . dice-test.scm

bench:
	guile --r7rs -L . -e bench dice-test.scm

plot:
	guile --r7rs -L . -e plot dice-test.scm
	gnuplot -e "plot 'n4v3d6.dat','n4v4d6.dat','n4v5d6.dat','mod2432-n4v4d6.dat','mod2432-n4v5d6.dat','mod1g-n4v4d6.dat','mod1g-n4v5d6.dat','modall-n4v5d6.dat' with points pt 3; pause mouse close"
