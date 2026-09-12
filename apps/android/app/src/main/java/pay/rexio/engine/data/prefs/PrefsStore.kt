package pay.rexio.engine.data.prefs

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStoreFile
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.MapSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json

data class AppPrefs(
    val serverUrl: String = DEFAULT_SERVER_URL,
    /** Epoch millis of the newest inbox SMS already scanned. */
    val inboxScanCursor: Long = 0,
    /** Epoch millis of the last successful SMS upload; 0 = never. */
    val lastSyncAt: Long = 0,
    /** sender_ids cache from GET /v1/device/config: provider → [display names]. */
    val senderIds: Map<String, List<String>> = emptyMap(),
    /** Epoch millis until which signed requests are paused after a 401. */
    val authCooldownUntil: Long = 0,
    /** Local SIM slot → profile mapping (slot is the 0-based OS index). */
    val simMapping: Map<Int, SimProfile> = emptyMap(),
) {
    companion object {
        const val DEFAULT_SERVER_URL = "https://api.pay.rexio.pro"
    }
}

/**
 * Non-secret app config. Pairing identity lives in [SessionStore] and the
 * device secret in SecretStore — never here. The file name is injectable so
 * tests get isolated stores.
 */
class PrefsStore private constructor(
    private val dataStore: DataStore<Preferences>,
    private val json: Json,
) {
    constructor(context: Context, fileName: String) : this(
        dataStore = PreferenceDataStoreFactory.create(
            scope = CoroutineScope(SupervisorJob() + Dispatchers.IO),
        ) { context.preferencesDataStoreFile(fileName) },
        json = Json { ignoreUnknownKeys = true },
    )

    private object Keys {
        val SERVER_URL = stringPreferencesKey("server_url")
        val INBOX_SCAN_CURSOR = longPreferencesKey("inbox_scan_cursor")
        val LAST_SYNC_AT = longPreferencesKey("last_sync_at")
        val SENDER_IDS = stringPreferencesKey("sender_ids_json")
        val AUTH_COOLDOWN_UNTIL = longPreferencesKey("auth_cooldown_until")
        val SIM_MAPPING = stringPreferencesKey("sim_mapping_json")
    }

    private val senderIdsSerializer = MapSerializer(
        String.serializer(),
        ListSerializer(String.serializer()),
    )

    private val simMappingSerializer = MapSerializer(
        Int.serializer(),
        SimProfile.serializer(),
    )

    val prefs: Flow<AppPrefs> = dataStore.data.map { p ->
        AppPrefs(
            serverUrl = p[Keys.SERVER_URL] ?: AppPrefs.DEFAULT_SERVER_URL,
            inboxScanCursor = p[Keys.INBOX_SCAN_CURSOR] ?: 0,
            lastSyncAt = p[Keys.LAST_SYNC_AT] ?: 0,
            senderIds = p[Keys.SENDER_IDS]?.let(::decodeSenderIds) ?: emptyMap(),
            authCooldownUntil = p[Keys.AUTH_COOLDOWN_UNTIL] ?: 0,
            simMapping = p[Keys.SIM_MAPPING]?.let(::decodeSimMapping) ?: emptyMap(),
        )
    }

    suspend fun setServerUrl(url: String) = dataStore.edit {
        it[Keys.SERVER_URL] = url.trimEnd('/')
    }

    suspend fun updateSenderIds(senderIds: Map<String, List<String>>) = dataStore.edit {
        it[Keys.SENDER_IDS] = json.encodeToString(senderIdsSerializer, senderIds)
    }

    suspend fun setInboxScanCursor(cursorMillis: Long) = dataStore.edit {
        it[Keys.INBOX_SCAN_CURSOR] = cursorMillis
    }

    suspend fun setLastSyncAt(millis: Long) = dataStore.edit {
        it[Keys.LAST_SYNC_AT] = millis
    }

    suspend fun setAuthCooldownUntil(millis: Long) = dataStore.edit {
        it[Keys.AUTH_COOLDOWN_UNTIL] = millis
    }

    suspend fun updateSimMapping(mapping: Map<Int, SimProfile>) = dataStore.edit {
        it[Keys.SIM_MAPPING] = json.encodeToString(simMappingSerializer, mapping)
    }

    /** Clears per-pairing state; the server URL override survives re-pairing. */
    suspend fun resetState() = dataStore.edit {
        it[Keys.INBOX_SCAN_CURSOR] = 0
        it[Keys.LAST_SYNC_AT] = 0
        it.remove(Keys.SENDER_IDS)
        it[Keys.AUTH_COOLDOWN_UNTIL] = 0
        it.remove(Keys.SIM_MAPPING)
    }

    private fun decodeSenderIds(raw: String): Map<String, List<String>> = runCatching {
        json.decodeFromString(senderIdsSerializer, raw)
    }.getOrDefault(emptyMap())

    private fun decodeSimMapping(raw: String): Map<Int, SimProfile> = runCatching {
        json.decodeFromString(simMappingSerializer, raw)
    }.getOrDefault(emptyMap())
}
