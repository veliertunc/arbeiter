import { SafeAreaView, StatusBar, StyleSheet } from "react-native";
import { AuthScreen } from "./src/screens/AuthScreen";

export default function App() {
  return <SafeAreaView style={styles.safe}>
    <StatusBar barStyle="dark-content"/>
    <AuthScreen />
  </SafeAreaView>;
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: "#f7f7fc" },
});
