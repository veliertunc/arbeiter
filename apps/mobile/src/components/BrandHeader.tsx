import { StyleSheet, Text, View } from "react-native";

export function BrandHeader() {
  return <View style={styles.row}>
    <View style={styles.mark}><Text style={styles.markText}>a</Text></View>
    <Text style={styles.wordmark}>arbeiter</Text>
    <View style={styles.environment}><View style={styles.dot}/><Text style={styles.envText}>SECURE IDENTITY</Text></View>
  </View>;
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center" },
  mark: { width: 30, height: 30, borderRadius: 9, backgroundColor: "#5b49dc", alignItems: "center", justifyContent: "center" },
  markText: { color: "white", fontSize: 19, fontWeight: "800" },
  wordmark: { fontSize: 18, fontWeight: "800", color: "#172133", marginLeft: 9 },
  environment: { marginLeft: "auto", flexDirection: "row", alignItems: "center", gap: 6 },
  envText: { fontSize: 8, color: "#8e98a8", letterSpacing: 1, fontWeight: "700" },
  dot: { width: 6, height: 6, borderRadius: 4, backgroundColor: "#25b982" },
});
