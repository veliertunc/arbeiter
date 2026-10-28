import { StyleSheet, Text, View } from "react-native";

export function SecurityNote() {
  return <View style={styles.note}>
    <Text style={styles.icon}>⌑</Text>
    <View><Text style={styles.title}>Your session stays private</Text><Text style={styles.copy}>Protected with encrypted device storage and secure cookies.</Text></View>
  </View>;
}

const styles = StyleSheet.create({
  note: { flexDirection: "row", alignItems: "center", gap: 12, marginTop: 24, padding: 14, backgroundColor: "#f0efff", borderRadius: 9 },
  icon: { width: 27, height: 27, textAlign: "center", textAlignVertical: "center", borderRadius: 8, overflow: "hidden", backgroundColor: "white", color: "#6251d7", fontSize: 17 },
  title: { fontSize: 10, color: "#384355", fontWeight: "700" },
  copy: { fontSize: 8, color: "#818b9a", marginTop: 3 },
});
