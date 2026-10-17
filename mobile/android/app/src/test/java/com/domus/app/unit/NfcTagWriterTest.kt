package com.domus.app.unit

import com.domus.app.services.nfc.NfcManagerMode
import com.domus.app.services.nfc.NfcTagManager
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

class NfcTagWriterTest {

    private lateinit var nfcManager: NfcTagManager

    @Before
    fun setUp() {
        nfcManager = NfcTagManager(context = null)
        nfcManager.setNfcAvailableForTest(true)
    }

    @Test
    fun testParseDomusUriActions() {
        val sceneTag = nfcManager.parseDomusUri("domus://scene/relax_evening")
        assertEquals("scene", sceneTag.actionType)
        assertEquals("relax_evening", sceneTag.targetId)

        val toggleTag = nfcManager.parseDomusUri("domus://action/cover.gate")
        assertEquals("toggle", toggleTag.actionType)
        assertEquals("cover.gate", toggleTag.targetId)

        val alarmTag = nfcManager.parseDomusUri("domus://alarm/disarm")
        assertEquals("alarm", alarmTag.actionType)
        assertEquals("disarm", alarmTag.targetId)
    }

    @Test
    fun testPrepareAndSimulateTagWrite() {
        val targetUri = "domus://action/cover.garage"
        var writtenResult: String? = null
        nfcManager.onTagWritten = { writtenResult = it }

        nfcManager.prepareTagWriting(targetUri)
        assertEquals(NfcManagerMode.WRITE, nfcManager.currentMode.value)
        assertEquals(targetUri, nfcManager.pendingWritePayload.value)

        val output = nfcManager.simulateTagWrite(targetUri)
        assertEquals(targetUri, output)
        assertEquals(targetUri, writtenResult)
        assertEquals(NfcManagerMode.READ, nfcManager.currentMode.value)
        assertNull(nfcManager.pendingWritePayload.value)
    }
}
