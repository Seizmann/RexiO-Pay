package pay.rexio.engine.core.time

interface Clock {
    fun nowMillis(): Long

    /** Unix seconds — the X-Rexio-Timestamp header unit. */
    fun nowSeconds(): Long = nowMillis() / 1000
}

object SystemClock : Clock {
    override fun nowMillis(): Long = System.currentTimeMillis()
}
