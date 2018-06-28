# chao

I've been getting `SQLITE_INTERRUPTED` errors immediately (not after a context is
cancelled), so I've been trying to reproduce it with chaos.

It seems like it worked!

```
2018/06/28 23:09:37 0 🔒 locked
2018/06/28 23:09:37 0 🔒 locked
2018/06/28 23:09:37 0 🔒 locked
2018/06/28 23:09:37 0 🔥 2.000522603s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.005098503s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.020954854s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.019908305s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.019117022s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.017959706s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.008870062s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.024383358s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.024319943s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.024309608s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.017645428s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.016851559s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.01672452s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.016644205s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.013748509s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.013740477s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🛑 24.954369ms after deadline (2.024954369s duration)
2018/06/28 23:09:37 0 🔥 2.018803372s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.018835777s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.019224017s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.018871256s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.018882081s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🔥 2.018906456s / 2s: we got a nil conn :o
2018/06/28 23:09:37 0 🛑 30.280279ms after deadline (2.030280279s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 967.77126ms before the deadline (2.03222874s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 965.597218ms before the deadline (2.034402782s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 963.254582ms before the deadline (2.036745418s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 959.592465ms before the deadline (2.040407535s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 958.748799ms before the deadline (2.041251201s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 955.71915ms before the deadline (2.04428085s duration)
2018/06/28 23:09:37 0 🛑 49.820261ms after deadline (2.049820261s duration)
2018/06/28 23:09:37 0 🛑 49.912799ms after deadline (2.049912799s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 928.109671ms before the deadline (2.071890329s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 876.25273ms before the deadline (2.12374727s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 867.362057ms before the deadline (2.132637943s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 862.551282ms before the deadline (2.137448718s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 860.399869ms before the deadline (2.139600131s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 858.487795ms before the deadline (2.141512205s duration)
2018/06/28 23:09:37 0 ⏰⏰⏰ we got interrupted 869.148068ms before the deadline (2.130851932s duration)
2018/06/28 23:09:37 1 ⏰⏰⏰ we got interrupted 1.133820017s before the deadline (1.866179983s duration)
```
