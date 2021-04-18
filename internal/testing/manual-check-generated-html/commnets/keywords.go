package comments

func foo() {
	if a := 1 == 1; a {
		/* ssssss */
	} else {
		// dafafasfa
	}
	if a := 1 == 1; a {
		/* ssssss */
	} else /*da faadsfaf /**/ /* /*dafafaf*/ // dafasfasfas
	//dafafasfasfa /*daafafaf/*
	{
		// dafafasfa
	}

	var x = []int{}
	for range x {
	}

	for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
	}
	for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
	}
	for range //aaa
	//aaaa
	x {
	}

	for range x {
	}

	for range /*da faadsfaf /**/ /* /*dafafaf*/ x {

		for range //aaa
		//aaaa
		x {
			for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
				for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
					if a := 1 == 1; a {
						/* ssssss */
					} else {
						// dafafasfa
					}
					if a := 1 == 1; a {
						/* ssssss */
					} else /*da faadsfaf /**/ /* /*dafafaf*/ // dafasfasfas
					//dafafasfasfa /*daafafaf/*
					{
						// dafafasfa
					}
				}
			}
		}

		for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
		}
		for range /*da faadsfaf /**/ /* /*dafafaf*/ x {
		}
	}

	for a := range x {
		_ = a
	}
}

func foo2() {
	if true {
	} else /*aaa*/ /*bbbb else*/ {
	}

	for range /*aaa*/ //bbb
	/*ccc*/ /*dddd*/
	//eeee	range
	[3]int{} {
	}

	switch func() interface{} {
		return interface{}(1)
	}().(type /*aaa*/ /*bbbb*/ //cccc
	//ddd type
	/*eee

	 */ //fff type
	/*ggg*/) {
	}
}

/*dddd*/ /*dddddd
 */ // aaaaa
